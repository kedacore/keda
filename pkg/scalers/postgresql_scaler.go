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
	_ "github.com/jackc/pgx/v5/stdlib" // PostreSQL driver required for this scaler
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
	metadata    postgreSQLMetadata
	connection  *sql.DB
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
}

type postgreSQLMetadata struct {
	Query                      string  `keda:"name=query,order=triggerMetadata"`
	TargetQueryValue           float64 `keda:"name=targetQueryValue,order=triggerMetadata"`
	ActivationTargetQueryValue float64 `keda:"name=activationTargetQueryValue,order=triggerMetadata,default=0"`
	Connection                 string  `keda:"name=connection,order=authParams;triggerMetadata, optional"`
	Host                       string  `keda:"name=host,order=authParams;triggerMetadata, optional"`
	Port                       string  `keda:"name=port,order=authParams;triggerMetadata, optional"`
	UserName                   string  `keda:"name=userName,order=authParams;triggerMetadata, optional"`
	Password                   string  `keda:"name=password,order=authParams;triggerMetadata, optional"`
	DBName                     string  `keda:"name=dbName,order=authParams;triggerMetadata, optional"`
	SSLMode                    string  `keda:"name=sslmode,order=authParams;triggerMetadata, optional"`

	TriggerIndex     int
	AzureAuthContext azureAuthContext
}

type azureAuthContext struct {
	cred  *azidentity.ChainedTokenCredential
	token *azcore.AccessToken
}

func (m *postgreSQLMetadata) Validate() error {
	if m.TargetQueryValue <= 0 {
		return fmt.Errorf("targetQueryValue must be greater than 0")
	}
	if m.Connection == "" && (m.Host == "" || m.Port == "" || m.UserName == "" || m.DBName == "") {
		return fmt.Errorf("either 'connection' or all of ('host', 'port', 'username', 'dbName', 'sslmode') must be provided")
	}
	return nil
}

func NewPostgreSQLScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "postgresql_scaler")

	meta, podIdentity, err := parsePostgreSQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing postgreSQL metadata: %w", err)
	}

	conn, err := getConnection(ctx, &meta, podIdentity, logger)
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

func parsePostgreSQLMetadata(config *scalersconfig.ScalerConfig) (postgreSQLMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := postgreSQLMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing postgresql metadata: %w", err)
	}

	if meta.Connection == "" && (meta.Host != "" && meta.Port != "" && meta.UserName != "" && meta.DBName != "") {
		meta.Connection = buildConnectionString(meta)
	}

	if err := meta.Validate(); err != nil {
		return meta, kedav1alpha1.AuthPodIdentity{}, err
	}

	meta.TriggerIndex = config.TriggerIndex
	podIdentity := config.PodIdentity

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if meta.Connection == "" {
			meta.Connection = buildConnectionString(meta)
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		cred, err := azure.NewChainedCredential(logr.Discard(), podIdentity)
		if err != nil {
			return meta, podIdentity, err
		}
		meta.AzureAuthContext.cred = cred
		meta.Connection = buildConnectionString(meta)
		meta.Connection += " password=%PASSWORD%"
	}

	return meta, podIdentity, nil
}

func buildConnectionString(meta postgreSQLMetadata) string {
	params := []string{
		"host=" + escapePostgreConnectionParameter(meta.Host),
		"port=" + escapePostgreConnectionParameter(meta.Port),
		"user=" + escapePostgreConnectionParameter(meta.UserName),
		"dbname=" + escapePostgreConnectionParameter(meta.DBName),
		"sslmode=" + escapePostgreConnectionParameter(meta.SSLMode),
	}
	if meta.Password != "" {
		params = append(params, "password="+escapePostgreConnectionParameter(meta.Password))
	}
	return strings.Join(params, " ")
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
		if s.metadata.AzureAuthContext.token.ExpiresOn.Before(time.Now()) {
			s.logger.Info("The Azure Access Token expired, retrieving a new Azure Access Token and instantiating a new Postgres connection object.")
			s.connection.Close()
			newConnection, err := getConnection(ctx, &s.metadata, s.podIdentity, s.logger)
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
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString("postgresql")),
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
	accessToken, err := metadata.AzureAuthContext.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{
			scope,
		},
	})
	if err != nil {
		return "", err
	}

	metadata.AzureAuthContext.token = &accessToken

	return metadata.AzureAuthContext.token.Token, nil
}
