package scalers

import (
	"context"
	"crypto/x509"
	"database/sql"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	rdsauth "github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib" // also registers the pgx driver used by sql.Open
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	// Azure AD resource ID for Azure Database for PostgreSQL is https://ossrdbms-aad.database.windows.net
	// https://learn.microsoft.com/en-us/azure/postgresql/single-server/how-to-connect-with-managed-identity
	azureDatabasePostgresResource = "https://ossrdbms-aad.database.windows.net/.default"
)

var (
	passwordConnPattern = regexp.MustCompile(`%PASSWORD%`)

	// defaultRDSRootCertPath is where the Amazon RDS CA bundle is expected when using
	// AWS IAM authentication and neither ca nor sslrootcert is set
	defaultRDSRootCertPath = "/etc/keda/certs/rds-global-bundle.pem"

	// buildRDSAuthToken signs an RDS IAM auth token, and it's a variable so tests can stub it
	buildRDSAuthToken = rdsauth.BuildAuthToken
)

type postgreSQLScaler struct {
	metricType  v2.MetricTargetType
	metadata    *postgreSQLMetadata
	connection  *sql.DB
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
}

type postgreSQLMetadata struct {
	TargetQueryValue           float64 `keda:"name=targetQueryValue,           order=triggerMetadata, optional"`
	ActivationTargetQueryValue float64 `keda:"name=activationTargetQueryValue, order=triggerMetadata, optional"`
	Connection                 string  `keda:"name=connection,                 order=authParams;resolvedEnv, optional"`
	Query                      string  `keda:"name=query,                      order=triggerMetadata"`
	triggerIndex               int
	azureAuthContext           azureAuthContext
	awsAuthorization           awsutils.AuthorizationMetadata
	rdsRootCAs                 *x509.CertPool

	Host     string `keda:"name=host,     order=authParams;triggerMetadata, optional"`
	Port     string `keda:"name=port,     order=authParams;triggerMetadata, optional"`
	UserName string `keda:"name=userName, order=authParams;triggerMetadata, optional"`
	DBName   string `keda:"name=dbName,   order=authParams;triggerMetadata, optional"`
	SslMode  string `keda:"name=sslmode,  order=authParams;triggerMetadata, optional"`

	SslRootCert string `keda:"name=sslrootcert, order=authParams;triggerMetadata, optional"`
	CA          string `keda:"name=ca,          order=authParams, optional"`
	AwsRegion   string `keda:"name=awsRegion,   order=triggerMetadata;authParams, optional"`

	Password string `keda:"name=password, order=authParams;resolvedEnv, optional"`
}

func (p *postgreSQLMetadata) Validate() error {
	if p.Connection == "" {
		if p.Host == "" {
			return fmt.Errorf("no host given")
		}

		if p.Port == "" {
			return fmt.Errorf("no port given")
		}

		if p.UserName == "" {
			return fmt.Errorf("no userName given")
		}

		if p.DBName == "" {
			return fmt.Errorf("no dbName given")
		}

		if p.SslMode == "" {
			return fmt.Errorf("no sslmode given")
		}
	}

	return nil
}

type azureAuthContext struct {
	cred  *azidentity.ChainedTokenCredential
	token *azcore.AccessToken
}

// NewPostgreSQLScaler creates a new postgreSQL scaler
func NewPostgreSQLScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "postgresql_scaler")

	meta, podIdentity, err := parsePostgreSQLMetadata(logger, config)
	if err != nil {
		return nil, fmt.Errorf("error parsing postgreSQL metadata: %w", err)
	}

	conn, err := getConnection(ctx, meta, podIdentity, logger)
	if err != nil {
		if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAws {
			awsutils.ClearAwsConfig(meta.awsAuthorization)
		}
		return nil, fmt.Errorf("error establishing postgreSQL connection: %w", err)
	}
	return &postgreSQLScaler{
		metricType:  metricType,
		metadata:    meta,
		connection:  conn,
		podIdentity: podIdentity,
		logger:      logger,
	}, nil
}

func parsePostgreSQLMetadata(logger logr.Logger, config *scalersconfig.ScalerConfig) (*postgreSQLMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := &postgreSQLMetadata{}
	authPodIdentity := kedav1alpha1.AuthPodIdentity{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, authPodIdentity, fmt.Errorf("error parsing postgresql metadata: %w", err)
	}

	if !config.AsMetricSource && meta.TargetQueryValue == 0 {
		return nil, authPodIdentity, fmt.Errorf("no targetQueryValue given")
	}

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if meta.Connection == "" {
			params := buildConnArray(meta)
			params = append(params, "password="+escapePostgreConnectionParameter(meta.Password))
			meta.Connection = strings.Join(params, " ")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		params := buildConnArray(meta)

		cred, err := azure.NewChainedCredential(logger, config.PodIdentity)
		if err != nil {
			return nil, authPodIdentity, err
		}
		meta.azureAuthContext.cred = cred
		authPodIdentity = kedav1alpha1.AuthPodIdentity{Provider: config.PodIdentity.Provider}

		params = append(params, "%PASSWORD%")
		meta.Connection = strings.Join(params, " ")
	case kedav1alpha1.PodIdentityProviderAws:
		if err := validateAwsRDSMetadata(meta); err != nil {
			return nil, authPodIdentity, err
		}

		awsAuthorization, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.AwsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
		if err != nil {
			return nil, authPodIdentity, err
		}

		// KEDA operator's identity must be explicitly opted into via identityOwner, do not silently use it
		if awsAuthorization.AwsRoleArn == "" && !isKedaIdentityOwner(config.PodIdentity) {
			return nil, authPodIdentity, fmt.Errorf(`aws pod identity requires either roleArn or identityOwner set to "keda" to use the KEDA operator's role`)
		}
		meta.awsAuthorization = awsAuthorization
		authPodIdentity = kedav1alpha1.AuthPodIdentity{Provider: config.PodIdentity.Provider}

		// Password is set per connection with a freshly signed token, see openAwsRDSConnection
		meta.Connection = strings.Join(buildConnArray(meta), " ")
	}
	meta.triggerIndex = config.TriggerIndex

	return meta, authPodIdentity, nil
}

func buildConnArray(meta *postgreSQLMetadata) []string {
	var params []string
	params = append(params, "host="+escapePostgreConnectionParameter(meta.Host))
	params = append(params, "port="+escapePostgreConnectionParameter(meta.Port))
	params = append(params, "user="+escapePostgreConnectionParameter(meta.UserName))
	params = append(params, "dbname="+escapePostgreConnectionParameter(meta.DBName))
	params = append(params, "sslmode="+escapePostgreConnectionParameter(meta.SslMode))
	if meta.SslRootCert != "" {
		params = append(params, "sslrootcert="+escapePostgreConnectionParameter(meta.SslRootCert))
	}

	return params
}

// validateAwsRDSMetadata checks that the trigger is configured for AWS RDS IAM authentication
// including a CA bundle to verify the server certificate: ca, sslrootcert, or the default RDS root certificate bundle
func validateAwsRDSMetadata(meta *postgreSQLMetadata) error {
	if meta.Connection != "" {
		return fmt.Errorf("connection is not supported with aws pod identity, set host, port, userName and dbName instead")
	}
	if meta.Password != "" {
		return fmt.Errorf("password must not be set with aws pod identity")
	}
	if strings.Contains(meta.Host, ",") {
		return fmt.Errorf("multiple hosts are not supported with aws pod identity")
	}
	if meta.AwsRegion == "" {
		return fmt.Errorf("no awsRegion given")
	}
	if meta.SslMode != "verify-full" {
		return fmt.Errorf("sslmode must be verify-full with aws pod identity, got %q", meta.SslMode)
	}
	if meta.CA != "" {
		if meta.SslRootCert != "" {
			return fmt.Errorf("ca and sslrootcert are mutually exclusive")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(meta.CA)) {
			return fmt.Errorf("ca does not contain any valid PEM encoded certificates")
		}
		meta.rdsRootCAs = pool
		return nil
	}
	if meta.SslRootCert == "" {
		meta.SslRootCert = defaultRDSRootCertPath
	}
	if _, err := os.Stat(meta.SslRootCert); err != nil {
		return fmt.Errorf("RDS CA bundle not found at %q, set ca or sslrootcert or mount the bundle at %q: %w", meta.SslRootCert, defaultRDSRootCertPath, err)
	}
	return nil
}

func isKedaIdentityOwner(podIdentity kedav1alpha1.AuthPodIdentity) bool {
	return podIdentity.IdentityOwner != nil && *podIdentity.IdentityOwner == "keda"
}

// openAwsRDSConnection opens a connection pool that signs an RDS IAM authentication token for every new connection
// Tokens are only checked when a connection is established, so open connections stay valid after their tokens expire
func openAwsRDSConnection(ctx context.Context, meta *postgreSQLMetadata) (*sql.DB, error) {
	connConfig, err := awsRDSConnConfig(meta)
	if err != nil {
		return nil, err
	}

	awsConfig, err := awsutils.GetAwsConfig(ctx, meta.awsAuthorization)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS config: %w", err)
	}

	endpoint := net.JoinHostPort(meta.Host, meta.Port)
	beforeConnect := func(ctx context.Context, cc *pgx.ConnConfig) error {
		token, err := buildRDSAuthToken(ctx, endpoint, meta.AwsRegion, meta.UserName, awsConfig.Credentials)
		if err != nil {
			return fmt.Errorf("error building RDS IAM authentication token: %w", err)
		}
		cc.Password = token
		return nil
	}

	return stdlib.OpenDB(*connConfig, stdlib.OptionBeforeConnect(beforeConnect)), nil
}

// awsRDSConnConfig parses the connection string and, if the ca auth param was provided, uses it to verify the server certificate
func awsRDSConnConfig(meta *postgreSQLMetadata) (*pgx.ConnConfig, error) {
	connConfig, err := pgx.ParseConfig(meta.Connection)
	if err != nil {
		return nil, fmt.Errorf("error parsing postgreSQL connection config: %w", err)
	}
	if meta.rdsRootCAs != nil {
		// sslmode=verify-full verifies the server certificate against RootCAs during the handshake
		for _, fallback := range connConfig.Fallbacks {
			if fallback.TLSConfig != nil {
				fallback.TLSConfig.RootCAs = meta.rdsRootCAs
			}
		}
		if connConfig.TLSConfig == nil {
			return nil, fmt.Errorf("TLS is not configured for the postgreSQL connection")
		}
		connConfig.TLSConfig.RootCAs = meta.rdsRootCAs
	}
	return connConfig, nil
}

func getConnection(ctx context.Context, meta *postgreSQLMetadata, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger) (*sql.DB, error) {
	connectionString := meta.Connection

	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		accessToken, err := getAzureAccessToken(ctx, meta, azureDatabasePostgresResource)
		if err != nil {
			return nil, err
		}
		newPasswordField := "password=" + escapePostgreConnectionParameter(accessToken)
		connectionString = passwordConnPattern.ReplaceAllString(meta.Connection, newPasswordField)
	}

	var db *sql.DB
	var err error
	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAws {
		db, err = openAwsRDSConnection(ctx, meta)
	} else {
		db, err = sql.Open("pgx", connectionString)
	}
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error opening postgreSQL: %s", err))
		return nil, err
	}
	err = db.PingContext(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error pinging postgreSQL: %s", err))
		return nil, err
	}
	return db, nil
}

// Close disposes of postgres connections
func (s *postgreSQLScaler) Close(context.Context) error {
	if s.podIdentity.Provider == kedav1alpha1.PodIdentityProviderAws {
		awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	}
	err := s.connection.Close()
	if err != nil {
		s.logger.Error(err, "Error closing postgreSQL connection")
		return err
	}
	return nil
}

func (s *postgreSQLScaler) getActiveNumber(ctx context.Context) (float64, error) {
	var id float64

	if s.podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		if s.metadata.azureAuthContext.token.ExpiresOn.Before(time.Now()) {
			s.logger.Info("The Azure Access Token expired, retrieving a new Azure Access Token and instantiating a new Postgres connection object.")
			s.connection.Close()
			newConnection, err := getConnection(ctx, s.metadata, s.podIdentity, s.logger)
			if err != nil {
				return 0, fmt.Errorf("error establishing postgreSQL connection: %w", err)
			}
			s.connection = newConnection
		}
	}

	err := s.connection.QueryRowContext(ctx, s.metadata.Query).Scan(&id)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("could not query postgreSQL: %s", err))
		return 0, fmt.Errorf("could not query postgreSQL: %w", err)
	}
	return id, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *postgreSQLScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString("postgresql")),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetQueryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *postgreSQLScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getActiveNumber(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting postgreSQL: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetQueryValue, nil
}

func escapePostgreConnectionParameter(str string) string {
	if strings.IndexFunc(str, unicode.IsSpace) == -1 && !strings.ContainsAny(str, `'\`) {
		return str
	}

	str = strings.ReplaceAll(str, `\`, `\\`)
	str = strings.ReplaceAll(str, "'", "\\'")
	return fmt.Sprintf("'%s'", str)
}

func getAzureAccessToken(ctx context.Context, metadata *postgreSQLMetadata, scope string) (string, error) {
	accessToken, err := metadata.azureAuthContext.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{
			scope,
		},
	})
	if err != nil {
		return "", err
	}

	metadata.azureAuthContext.token = &accessToken

	return metadata.azureAuthContext.token.Token, nil
}
