package resolver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/aws"
	"github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/azure"
	"github.com/akeylesslabs/akeyless-go-cloud-id/cloudprovider/gcp"
	"github.com/akeylesslabs/akeyless-go/v5"
	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	authAccessKey              = "access_key"
	authAwsIam                 = "aws_iam"
	authK8s                    = "k8s"
	authGcp                    = "gcp"
	authAzureAd                = "azure_ad"
	publicGatewayURL           = "https://api.akeyless.io"
	userAgent                  = "keda.sh"
	staticSecretResponse       = "STATIC_SECRET"
	dynamicSecretResponse      = "DYNAMIC_SECRET"
	rotatedSecretResponse      = "ROTATED_SECRET"
	allSecretTypes             = "all"
	clientSource               = "akeylessclienttype"
	k8sServiceAccountTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var supportedSecretTypes = []string{staticSecretResponse, dynamicSecretResponse, rotatedSecretResponse}

type AkeylessHandler struct {
	akeyless *kedav1alpha1.Akeyless
	client   *akeyless.V2ApiService
	token    string
	logger   logr.Logger
}

// Initialize the AkeylessHandler
func (h *AkeylessHandler) Initialize(ctx context.Context) error {
	// Validate Gateway URL is not empty and is a valid URL
	if h.akeyless.GatewayURL == "" {
		h.logger.Info(fmt.Sprintf("gatewayUrl is not set, using default value %s...", publicGatewayURL))
		h.akeyless.GatewayURL = publicGatewayURL
	} else {
		url, err := url.ParseRequestURI(h.akeyless.GatewayURL)
		if err != nil {
			return errors.New("invalid gateway URL '" + h.akeyless.GatewayURL + "': " + err.Error())
		}

		// if the path is empty, add the v2 API path
		if url.Path == "" {
			h.logger.Info(fmt.Sprintf("gatewayUrl path is empty, adding default v2 API path (%s)", "/api/v2"))
			url.Path = "/api/v2"
		}

		h.akeyless.GatewayURL = url.String()
		h.logger.Info(fmt.Sprintf("gatewayUrl set to '%s'", h.akeyless.GatewayURL))
	}

	// Validate Access ID
	if h.akeyless.AccessID == "" {
		return errors.New("accessId is required")
	}

	h.logger.Info(fmt.Sprintf("initializing Akeyless handler '%s'...", h.akeyless.GatewayURL))
	err := h.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("unable to authenticate with Akeyless: %w", err)
	}

	h.logger.Info("Akeyless handler initialized successfully")
	return nil
}

// Authenticate with Akeyless
func (h *AkeylessHandler) Authenticate(ctx context.Context) error {
	h.logger.Info(fmt.Sprintf("authenticating with Akeyless '%s' using Access ID '%s'...", h.akeyless.GatewayURL, h.akeyless.AccessID))
	authRequest := akeyless.NewAuth()
	authRequest.SetAccessId(h.akeyless.AccessID)

	// Get the authentication method
	h.logger.Info("extracting access type from Access ID...")
	accessTypeChar, err := extractAccessTypeChar(h.akeyless.AccessID)
	if err != nil {
		return errors.New("unable to extract access type character from accessId, expected format is p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12})")
	}
	accessType, err := getAccessTypeDisplayName(accessTypeChar)
	if err != nil {
		return errors.New("unable to get access type display name, expected format is p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12})")
	}

	h.logger.Info(fmt.Sprintf("authenticating using access type '%s'...", accessType))

	switch accessType {
	case authAccessKey:
		accessKey := h.akeyless.AccessKey
		if accessKey == nil {
			return errors.New("access key is required for access type 'access_key'")
		}
		authRequest.SetAccessKey(*accessKey)
	case authAwsIam:
		authRequest.SetAccessType(authAwsIam)
		id, err := aws.GetCloudId()
		if err != nil {
			return fmt.Errorf("unable to get cloud ID for AWS IAM: %w", err)
		}
		authRequest.SetCloudId(id)
	case authGcp:
		authRequest.SetAccessType(authGcp)
		// TODO add conf for audience
		id, err := gcp.GetCloudID("akeyless.io")
		if err != nil {
			return fmt.Errorf("unable to get cloud ID for GCP: %w", err)
		}
		authRequest.SetCloudId(id)
	case authAzureAd:
		authRequest.SetAccessType(authAzureAd)
		// TODO add conf for object ID
		id, err := azure.GetCloudId("")
		if err != nil {
			return fmt.Errorf("unable to get cloud ID for Azure AD: %w", err)
		}
		authRequest.SetCloudId(id)
	case authK8s:
		authRequest.SetAccessType(authK8s)

		if h.akeyless.K8sAuthConfigName == "" {
			return errors.New("k8sAuthConfigName is required for access type 'k8s'")
		}
		authRequest.SetK8sAuthConfigName(h.akeyless.K8sAuthConfigName)

		if h.akeyless.K8sGatewayURL == "" {
			h.logger.Info(fmt.Sprintf("k8sGatewayUrl is not provided, using gatewayUrl '%s'...", h.akeyless.GatewayURL))
			h.akeyless.K8sGatewayURL = h.akeyless.GatewayURL
		}
		h.akeyless.K8sGatewayURL = strings.TrimSuffix(h.akeyless.K8sGatewayURL, "/api/v2")
		authRequest.SetGatewayUrl(h.akeyless.K8sGatewayURL)

		if h.akeyless.K8sServiceAccountToken == "" {
			h.logger.Info("k8sServiceAccountToken is not provided, attempting to retrieve from file...")
			token, err := os.ReadFile(k8sServiceAccountTokenFile)
			if err != nil {
				h.logger.Info(fmt.Sprintf("unable to read k8s service account token from file '%s': %s", k8sServiceAccountTokenFile, err.Error()))
				return errors.New("unable to read k8s service account token from file '" + k8sServiceAccountTokenFile + "': " + err.Error())
			}
			h.akeyless.K8sServiceAccountToken = string(token)
			h.logger.Info(fmt.Sprintf("k8s service account token retrieved from file '%s'", k8sServiceAccountTokenFile))
		}

		// base64 encode the token if it's not already encoded
		if _, err := base64.StdEncoding.DecodeString(h.akeyless.K8sServiceAccountToken); err != nil {
			h.logger.Info("k8sServiceAccountToken is not base64 encoded, encoding it...")
			h.akeyless.K8sServiceAccountToken = base64.StdEncoding.EncodeToString([]byte(h.akeyless.K8sServiceAccountToken))
		}

		authRequest.SetK8sServiceAccountToken(h.akeyless.K8sServiceAccountToken)

	default:
		return fmt.Errorf("unsupported access type: %s", accessType)
	}

	// Create Akeyless API client configuration
	// TODO add support for TLS
	h.logger.Info("creating Akeyless API client configuration...")
	config := akeyless.NewConfiguration()
	config.Servers = []akeyless.ServerConfiguration{
		{
			URL: h.akeyless.GatewayURL,
		},
	}
	config.UserAgent = userAgent
	config.AddDefaultHeader(clientSource, userAgent)

	h.client = akeyless.NewAPIClient(config).V2Api

	h.logger.Info("authenticating with Akeyless...")
	out, httpResponse, err := h.client.Auth(ctx).Body(*authRequest).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to authenticate with Akeyless API: %w", err)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return fmt.Errorf("failed to authenticate with Akeyless API (HTTP status code: %d): %w", httpResponse.StatusCode, errors.New(httpResponse.Status))
	}

	h.logger.Info(fmt.Sprintf("authentication successful - token expires at %s", out.GetExpiration()))
	h.token = out.GetToken()
	return nil
}

func (h *AkeylessHandler) GetSecretsValue(ctx context.Context, secretResults map[string]string) (map[string]string, error) {
	h.logger.Info(fmt.Sprintf("getting secrets values for %d secrets...", len(h.akeyless.Secrets)))
	for _, secret := range h.akeyless.Secrets {
		secretType, err := h.GetSecretType(ctx, secret.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret type for '%s': %w", secret.Path, err)
		}

		if !slices.Contains(supportedSecretTypes, secretType) {
			return nil, fmt.Errorf("unsupported secret type '%s' for '%s': supported secret types are: %s", secretType, secret.Path, strings.Join(supportedSecretTypes, ", "))
		}

		var secretValue string
		switch secretType {
		case staticSecretResponse:
			secretValue, err = h.getStaticSecretValue(ctx, secret)
		case dynamicSecretResponse:
			secretValue, err = h.getDynamicSecretValue(ctx, secret)
		case rotatedSecretResponse:
			secretValue, err = h.getRotatedSecretValue(ctx, secret)
		}

		if err != nil {
			return nil, err
		}

		secretResults[secret.Parameter] = secretValue
	}
	h.logger.Info(fmt.Sprintf("returning %d secrets values", len(secretResults)))
	return secretResults, nil
}

// getStaticSecretValue handles getting static secret values
func (h *AkeylessHandler) getStaticSecretValue(ctx context.Context, secret kedav1alpha1.AkeylessSecret) (string, error) {
	h.logger.Info(fmt.Sprintf("getting secret value for static secret '%s'...", secret.Path))
	getSecretValue := akeyless.NewGetSecretValue([]string{secret.Path})
	getSecretValue.SetToken(h.token)
	secretRespMap, httpResponse, apiErr := h.client.GetSecretValue(ctx).Body(*getSecretValue).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if apiErr != nil {
		return "", fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: %w", secret.Path, apiErr)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return "", fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API (HTTP status code: %d): %w", secret.Path, httpResponse.StatusCode, errors.New(httpResponse.Status))
	}

	secretValueStr, ok := secretRespMap[secret.Path].(string)
	if !ok {
		return "", fmt.Errorf("failed to get secret '%s' value for static secret from Akeyless API: %w", secret.Path, apiErr)
	}

	return h.parseStaticSecretValue(secret.Path, secret.Key, secretValueStr)
}

// parseStaticSecretValue parses static secret value, handling JSON if needed
func (h *AkeylessHandler) parseStaticSecretValue(path, key, secretValueStr string) (string, error) {
	var jsonMap map[string]any
	if err := json.Unmarshal([]byte(secretValueStr), &jsonMap); err == nil {
		// Successfully parsed as JSON
		h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a JSON string", path))

		if key == "" {
			h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a JSON string and key is not provided, returning stringified JSON", path))
			jsonBytes, err := json.Marshal(jsonMap)
			if err != nil {
				return "", fmt.Errorf("failed to marshal JSON value for static secret '%s': %w", path, err)
			}
			return string(jsonBytes), nil
		}

		h.logger.Info(fmt.Sprintf("secret value for static secret '%s' is a JSON string and key is provided, searching for value for key '%s'", path, key))
		secretValueStr, found := jsonMap[key].(string)
		if !found {
			return "", fmt.Errorf("failed to get secret '%s' value for static secret: key '%s' not found", path, key)
		}
		return secretValueStr, nil
	}
	return secretValueStr, nil
}

// getDynamicSecretValue handles getting dynamic secret values
func (h *AkeylessHandler) getDynamicSecretValue(ctx context.Context, secret kedav1alpha1.AkeylessSecret) (string, error) {
	h.logger.Info(fmt.Sprintf("getting dynamic secret value for '%s' from Akeyless API...", secret.Path))
	getDynamicSecretValue := akeyless.NewGetDynamicSecretValue(secret.Path)
	getDynamicSecretValue.SetToken(h.token)
	secretRespMap, httpResponse, apiErr := h.client.GetDynamicSecretValue(ctx).Body(*getDynamicSecretValue).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if apiErr != nil {
		return "", fmt.Errorf("failed to get secret '%s' value for dynamic secret from Akeyless API: %w", secret.Path, apiErr)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return "", fmt.Errorf("failed to get secret '%s' value for dynamic secret from Akeyless API (HTTP status code: %d): %w", secret.Path, httpResponse.StatusCode, errors.New(httpResponse.Status))
	}

	var dynamicSecretResp struct {
		Value string `json:"value"`
		Error string `json:"error"`
	}

	if err := mapstructure.Decode(secretRespMap, &dynamicSecretResp); err != nil {
		return "", fmt.Errorf("dynamic secret '%s' response in unexpected format: %w", secret.Path, err)
	}

	if dynamicSecretResp.Value == "" {
		return "", fmt.Errorf("dynamic secret '%s' response contains no value", secret.Path)
	}

	if dynamicSecretResp.Error != "" {
		return "", fmt.Errorf("dynamic secret '%s' response contains an error: %s", secret.Path, dynamicSecretResp.Error)
	}

	var mapValue map[string]string
	if err := json.Unmarshal([]byte(dynamicSecretResp.Value), &mapValue); err != nil {
		return "", fmt.Errorf("dynamic secret '%s' value in unexpected format: %w", secret.Path, err)
	}

	return h.extractSecretValueFromMap(secret.Path, secret.Key, mapValue, "dynamic")
}

// getRotatedSecretValue handles getting rotated secret values
func (h *AkeylessHandler) getRotatedSecretValue(ctx context.Context, secret kedav1alpha1.AkeylessSecret) (string, error) {
	h.logger.Info(fmt.Sprintf("getting rotated secret value for '%s'...", secret.Path))
	getRotatedSecretValue := akeyless.NewGetRotatedSecretValue(secret.Path)
	getRotatedSecretValue.SetToken(h.token)
	secretRespMap, httpResponse, apiErr := h.client.GetRotatedSecretValue(ctx).Body(*getRotatedSecretValue).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if apiErr != nil {
		return "", fmt.Errorf("failed to get secret '%s' value for rotated secret from Akeyless API: %w", secret.Path, apiErr)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return "", fmt.Errorf("failed to get secret '%s' value for rotated secret from Akeyless API (HTTP status code: %d): %w", secret.Path, httpResponse.StatusCode, errors.New(httpResponse.Status))
	}

	var rotatedSecretResponse struct {
		Value map[string]string `json:"value"`
	}

	if err := mapstructure.Decode(secretRespMap, &rotatedSecretResponse); err != nil {
		return "", fmt.Errorf("rotated secret '%s' response in unexpected format: %w", secret.Path, err)
	}

	if rotatedSecretResponse.Value == nil {
		return "", fmt.Errorf("rotated secret '%s' response contains no value", secret.Path)
	}

	return h.extractSecretValueFromMap(secret.Path, secret.Key, rotatedSecretResponse.Value, "rotated")
}

// extractSecretValueFromMap extracts a value from a map, handling key presence/absence
func (h *AkeylessHandler) extractSecretValueFromMap(path, key string, mapValue map[string]string, secretType string) (string, error) {
	if key == "" {
		h.logger.Info(fmt.Sprintf("%s secret value for '%s' is a map[string]string but key is not provided, returning stringified JSON", secretType, path))
		jsonBytes, err := json.Marshal(mapValue)
		if err != nil {
			return "", fmt.Errorf("%s secret '%s' value in unexpected format: %w", secretType, path, err)
		}
		return string(jsonBytes), nil
	}

	h.logger.Info(fmt.Sprintf("%s secret value for '%s' is a map[string]string and key is provided, returning value for key '%s'", secretType, path, key))
	secretValue, found := mapValue[key]
	if !found {
		return "", fmt.Errorf("failed to get secret '%s' value for %s secret: key '%s' not found", path, secretType, key)
	}
	return secretValue, nil
}

func (h *AkeylessHandler) GetSecretType(ctx context.Context, secretName string) (string, error) {
	describeItem := akeyless.NewDescribeItem(secretName)
	describeItem.SetToken(h.token)
	describeItemResp, httpResponse, apiErr := h.client.DescribeItem(ctx).Body(*describeItem).Execute()
	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
	if apiErr != nil {
		return "", fmt.Errorf("failed to describe item '%s' from Akeyless API: %w", secretName, apiErr)
	}
	if httpResponse != nil && httpResponse.StatusCode != 200 {
		return "", fmt.Errorf("failed to describe item '%s' from Akeyless API (HTTP status code: %d): %w", secretName, httpResponse.StatusCode, errors.New(httpResponse.Status))
	}

	if describeItemResp.ItemType == nil {
		return "", errors.New("unable to retrieve secret type, missing type in describe item response")
	}

	return *describeItemResp.ItemType, nil
}

// AccessTypeCharMap maps single-character access types to their display names.
var accessTypeCharMap = map[string]string{
	"a": authAccessKey,
	"w": authAwsIam,
	"k": authK8s,
	"g": authGcp,
	"z": authAzureAd,
}

// AccessIdRegex is the compiled regular expression for validating Akeyless Access IDs.
var accessIDRegex = regexp.MustCompile(`^p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12})$`)

// isValidAccessIDFormat validates the format of an Akeyless Access ID.
// The format is p-([A-Za-z0-9]{14}|[A-Za-z0-9]{12}).
// It returns true if the format is valid, and false otherwise.
func isValidAccessIDFormat(accessID string) bool {
	return accessIDRegex.MatchString(accessID)
}

// extractAccessTypeChar extracts the Akeyless Access Type character from a valid Access ID.
// The access type character is the second to last character of the ID part.
// It returns the single-character access type (e.g., 'a', 'o') or an empty string and an error if the format is invalid.
func extractAccessTypeChar(accessID string) (string, error) {
	if !isValidAccessIDFormat(accessID) {
		return "", errors.New("invalid access ID format")
	}
	parts := strings.Split(accessID, "-")
	idPart := parts[1] // Get the part after "p-"
	// The access type char is the second-to-last character
	return string(idPart[len(idPart)-2]), nil
}

// getAccessTypeDisplayName gets the full display name of the access type from the character.
// It returns the display name (e.g., 'api_key') or an error if the type character is unknown.
func getAccessTypeDisplayName(typeChar string) (string, error) {
	if typeChar == "" {
		return "", errors.New("unable to retrieve access type, missing type char")
	}
	displayName, ok := accessTypeCharMap[typeChar]
	if !ok {
		return "Unknown", errors.New("access type character not found in map")
	}
	return displayName, nil
}
