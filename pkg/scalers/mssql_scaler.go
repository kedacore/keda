package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/go-logr/logr"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	azureDatabaseMSSQLResource = "https://database.windows.net/.default"
)

type mssqlScaler struct {
	metricType  v2.MetricTargetType
	metadata    *mssqlMetadata
	connection  *sql.DB
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
}

type mssqlMetadata struct {
	ConnectionString      string  `keda:"name=connectionString,      order=authParams;resolvedEnv, optional"`
	Username              string  `keda:"name=username,              order=authParams;triggerMetadata, optional"`
	Password              string  `keda:"name=password,              order=authParams;resolvedEnv, optional"`
	Host                  string  `keda:"name=host,                  order=authParams;triggerMetadata, optional"`
	Port                  int     `keda:"name=port,                  order=authParams;triggerMetadata, optional"`
	Database              string  `keda:"name=database,              order=authParams;triggerMetadata, optional"`
	Query                 string  `keda:"name=query,                 order=triggerMetadata"`
	TargetValue           float64 `keda:"name=targetValue,           order=triggerMetadata"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`

	TriggerIndex     int
	azureAuthContext azureAuthContext
}

func (m *mssqlMetadata) Validate() error {
	if m.ConnectionString == "" && m.Host == "" {
		return fmt.Errorf("must provide either connectionstring or host")
	}
	return nil
}

func getMSSQLAzureAccessToken(ctx context.Context, metadata *mssqlMetadata, scope string) (string, error) {
	accessToken, err := metadata.azureAuthContext.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return "", err
	}
	metadata.azureAuthContext.token = &accessToken
	return accessToken.Token, nil
}

func NewMSSQLScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "mssql_scaler")

	meta, podIdentity, err := parseMSSQLMetadata(logger, config)
	if err != nil {
		return nil, err
	}

	scaler := &mssqlScaler{
		metricType:  metricType,
		metadata:    meta,
		podIdentity: podIdentity,
		logger:      logger,
	}

	conn, err := newMSSQLConnection(ctx, scaler)
	if err != nil {
		return nil, fmt.Errorf("error establishing mssql connection: %w", err)
	}

	scaler.connection = conn

	return scaler, nil
}

func parseMSSQLMetadata(logger logr.Logger, config *scalersconfig.ScalerConfig) (*mssqlMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := &mssqlMetadata{}
	authPodIdentity := kedav1alpha1.AuthPodIdentity{}
	meta.TriggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, authPodIdentity, err
	}

	if !config.AsMetricSource && meta.TargetValue == 0 {
		return nil, authPodIdentity, fmt.Errorf("no targetValue given")
	}

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// existing behavior — no changes needed
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		cred, err := azure.NewChainedCredential(logger, config.PodIdentity)
		if err != nil {
			return nil, authPodIdentity, err
		}
		meta.azureAuthContext.cred = cred
		authPodIdentity = kedav1alpha1.AuthPodIdentity{Provider: config.PodIdentity.Provider}
	default:
		return nil, authPodIdentity, fmt.Errorf("pod identity %s is not supported for mssql scaler", config.PodIdentity.Provider)
	}

	return meta, authPodIdentity, nil
}

func newMSSQLConnection(ctx context.Context, s *mssqlScaler) (*sql.DB, error) {
	if s.podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		return newMSSQLWorkloadIdentityConnection(ctx, s)
	}

	connStr := getMSSQLConnectionString(s)

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		s.logger.Error(err, "Found error opening mssql")
		return nil, err
	}

	err = db.PingContext(ctx)
	if err != nil {
		s.logger.Error(err, "Found error pinging mssql")
		return nil, err
	}

	return db, nil
}

func newMSSQLWorkloadIdentityConnection(ctx context.Context, s *mssqlScaler) (*sql.DB, error) {
	connStr := getMSSQLConnectionString(s)

	dsnConfig, err := msdsn.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing mssql DSN: %w", err)
	}

	tokenProvider := func(ctx context.Context) (string, error) {
		return getMSSQLAzureAccessToken(ctx, s.metadata, azureDatabaseMSSQLResource)
	}

	connector, err := mssql.NewSecurityTokenConnector(dsnConfig, tokenProvider)
	if err != nil {
		return nil, fmt.Errorf("error creating mssql security token connector: %w", err)
	}

	db := sql.OpenDB(connector)
	err = db.PingContext(ctx)
	if err != nil {
		s.logger.Error(err, "Found error pinging mssql with workload identity")
		return nil, err
	}

	return db, nil
}

func getMSSQLConnectionString(s *mssqlScaler) string {
	meta := s.metadata
	if meta.ConnectionString != "" {
		return meta.ConnectionString
	}

	query := url.Values{}
	if meta.Database != "" {
		query.Add("database", meta.Database)
	}

	connectionURL := &url.URL{Scheme: "sqlserver", RawQuery: query.Encode()}
	if meta.Username != "" {
		if meta.Password != "" {
			connectionURL.User = url.UserPassword(meta.Username, meta.Password)
		} else {
			connectionURL.User = url.User(meta.Username)
		}
	}

	if meta.Port > 0 {
		connectionURL.Host = net.JoinHostPort(meta.Host, fmt.Sprintf("%d", meta.Port))
	} else {
		connectionURL.Host = meta.Host
	}

	return connectionURL.String()
}

func (s *mssqlScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, "mssql"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

func (s *mssqlScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting mssql: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetValue, nil
}

func (s *mssqlScaler) getQueryResult(ctx context.Context) (float64, error) {
	if s.podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		if s.metadata.azureAuthContext.token != nil && s.metadata.azureAuthContext.token.ExpiresOn.Before(time.Now()) {
			s.logger.Info("Azure access token expired, reconnecting to MSSQL")
			s.connection.Close()
			newConnection, err := newMSSQLConnection(ctx, s)
			if err != nil {
				return 0, fmt.Errorf("error establishing mssql connection: %w", err)
			}
			s.connection = newConnection
		}
	}

	var value float64

	err := s.connection.QueryRowContext(ctx, s.metadata.Query).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		value = 0
	case err != nil:
		s.logger.Error(err, fmt.Sprintf("Could not query mssql database: %s", err))
		return 0, err
	}

	return value, nil
}

func (s *mssqlScaler) Close(context.Context) error {
	err := s.connection.Close()
	if err != nil {
		s.logger.Error(err, "Error closing mssql connection")
		return err
	}

	return nil
}
