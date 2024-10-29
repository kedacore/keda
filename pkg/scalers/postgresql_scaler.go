package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	_ "github.com/jackc/pgx/v5/stdlib" // PostreSQL drive required for this scaler
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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
)

type postgreSQLScaler struct {
	metricType  v2.MetricTargetType
	metadata    *postgreSQLMetadata
	connection  *sql.DB
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
}

type postgreSQLMetadata struct {
	TargetQueryValue           float64 `keda:"name=targetQueryValue, order=triggerMetadata, optional"`
	ActivationTargetQueryValue float64 `keda:"name=activationTargetQueryValue, order=triggerMetadata, optional"`
	Connection                 string  `keda:"name=connection, order=authParams;resolvedEnv, optional"`
	Query                      string  `keda:"name=query, order=triggerMetadata"`
	triggerIndex               int
	azureAuthContext           azureAuthContext

	Host     string `keda:"name=host, order=authParams;triggerMetadata, optional"`
	Port     string `keda:"name=port, order=authParams;triggerMetadata, optional"`
	UserName string `keda:"name=userName, order=authParams;triggerMetadata, optional"`
	DBName   string `keda:"name=dbName, order=authParams;triggerMetadata, optional"`
	SslMode  string `keda:"name=sslmode, order=authParams;triggerMetadata, optional"`

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

	return params
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

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error opening postgreSQL: %s", err))
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error pinging postgreSQL: %s", err))
		return nil, err
	}
	return db, nil
}

// Close disposes of postgres connections
func (s *postgreSQLScaler) Close(context.Context) error {
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
	if !strings.Contains(str, " ") {
		return str
	}

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
