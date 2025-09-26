package scalers

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	// AWS IAM authentication token validity period (tokens are valid for 15 minutes)
	// We refresh at 75% of the validity period to provide sufficient buffer for network issues
	awsIAMTokenValidityPeriod = 15 * time.Minute
	awsIAMTokenRefreshPercent = 0.75 // Refresh when 75% of validity period has passed
)

type awsIAMAuthContext struct {
	token     string
	expiresAt time.Time
}

// isRDSHost checks if the host appears to be an RDS endpoint
func isRDSHost(host string) bool {
	if host == "" {
		return false
	}

	// Check for comprehensive RDS endpoint patterns
	// RDS endpoints follow the pattern: <instance-id>.<unique-string>.<region>.rds.amazonaws.com
	// We support various AWS partitions and service variations
	rdsPatterns := []string{
		// Standard AWS partitions
		".rds.amazonaws.com",        // Standard regions
		".rds.amazonaws.com.cn",     // China regions

		// FIPS endpoints
		".rds-fips.amazonaws.com",   // FIPS standard regions

		// GovCloud regions
		".rds.us-gov-east-1.amazonaws.com",
		".rds.us-gov-west-1.amazonaws.com",

		// New API endpoints (some regions support both formats)
		".rds.api.aws",
		".rds-fips.api.aws",
	}

	for _, pattern := range rdsPatterns {
		if strings.Contains(host, pattern) {
			return true
		}
	}

	return false
}

// hasAWSIRSA checks if AWS IRSA credentials are available
func hasAWSIRSA() bool {
	// Check for common IRSA environment variables
	// These are set when a pod uses a service account with an IAM role
	webIdentityFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	roleArn := os.Getenv("AWS_ROLE_ARN")

	return webIdentityFile != "" && roleArn != ""
}

// getAWSRegionFromHost extracts the AWS region from an RDS endpoint
func getAWSRegionFromHost(host string) string {
	// RDS endpoints typically follow pattern: instance.cluster-id.region.rds.amazonaws.com
	parts := strings.Split(host, ".")

	// Find the index of "rds" and get the region before it
	for i, part := range parts {
		if part == "rds" && i > 0 {
			return parts[i-1]
		}
	}

	// Check environment variable
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	// Default to us-east-1 if we can't determine the region
	return "us-east-1"
}

// generateRDSIAMToken generates an AWS RDS IAM authentication token
func generateRDSIAMToken(ctx context.Context, host, port, username string, logger logr.Logger) (string, error) {
	region := getAWSRegionFromHost(host)

	// Load AWS config with IRSA credentials
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Build the RDS endpoint
	endpoint := net.JoinHostPort(host, port)

	// Generate the IAM auth token
	token, err := auth.BuildAuthToken(ctx, endpoint, region, username, cfg.Credentials)
	if err != nil {
		return "", fmt.Errorf("failed to generate RDS IAM token: %w", err)
	}

	logger.V(1).Info("Generated AWS RDS IAM authentication token",
		"host", host,
		"port", port,
		"username", username,
		"region", region)

	return token, nil
}

// shouldUseAWSIAM determines if AWS IAM authentication should be used
func shouldUseAWSIAM(meta *postgreSQLMetadata, podIdentity kedav1alpha1.AuthPodIdentity) bool {
	// Use AWS IAM if:
	// 1. No password is provided
	// 2. Host appears to be RDS
	// 3. AWS IRSA is available
	// 4. Not already using Azure Workload Identity

	if meta.Password != "" {
		return false
	}

	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		return false
	}

	if !isRDSHost(meta.Host) {
		return false
	}

	if !hasAWSIRSA() {
		return false
	}

	return true
}

// refreshAWSIAMTokenIfNeeded checks if the AWS IAM token needs refresh and refreshes it
func refreshAWSIAMTokenIfNeeded(ctx context.Context, meta *postgreSQLMetadata, logger logr.Logger) (bool, error) {
	if meta.awsIAMContext == nil {
		return false, nil
	}

	// Calculate refresh time based on percentage of validity period
	tokenAge := time.Since(meta.awsIAMContext.expiresAt.Add(-awsIAMTokenValidityPeriod))
	refreshThreshold := time.Duration(float64(awsIAMTokenValidityPeriod) * awsIAMTokenRefreshPercent)

	if tokenAge < refreshThreshold {
		return false, nil // Token still within acceptable age
	}

	logger.Info("AWS RDS IAM token needs refresh based on age threshold",
		"tokenAge", tokenAge,
		"refreshThreshold", refreshThreshold)

	// Generate new token
	token, err := generateRDSIAMToken(ctx, meta.Host, meta.Port, meta.UserName, logger)
	if err != nil {
		return false, fmt.Errorf("failed to refresh RDS IAM token: %w", err)
	}

	// Update token context
	meta.awsIAMContext.token = token
	meta.awsIAMContext.expiresAt = time.Now().Add(awsIAMTokenValidityPeriod)

	// Update connection string with new token
	params := buildConnArray(meta)
	params = append(params, "password="+escapePostgreConnectionParameter(token))
	meta.Connection = strings.Join(params, " ")

	return true, nil
}

// setupAWSIAMAuth configures AWS IAM authentication for the PostgreSQL connection
func setupAWSIAMAuth(ctx context.Context, meta *postgreSQLMetadata, logger logr.Logger) error {
	// Generate initial token
	token, err := generateRDSIAMToken(ctx, meta.Host, meta.Port, meta.UserName, logger)
	if err != nil {
		return fmt.Errorf("failed to generate initial RDS IAM token: %w", err)
	}

	// Store token metadata for refresh checks
	meta.awsIAMContext = &awsIAMAuthContext{
		token:     token,
		expiresAt: time.Now().Add(awsIAMTokenValidityPeriod),
	}

	// Build connection string with IAM token as password
	params := buildConnArray(meta)
	params = append(params, "password="+escapePostgreConnectionParameter(token))
	meta.Connection = strings.Join(params, " ")

	logger.Info("Configured AWS RDS IAM authentication for PostgreSQL connection",
		"host", meta.Host,
		"username", meta.UserName)

	return nil
}