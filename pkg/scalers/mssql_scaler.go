package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"

	// mssql driver required for this scaler
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type mssqlScaler struct {
	metricType v2.MetricTargetType
	metadata   mssqlMetadata
	connection *sql.DB
	logger     logr.Logger
	azureOAuth *azure.ADWorkloadIdentityTokenProvider
}

type mssqlMetadata struct {
	ConnectionString      string  `keda:"name=connectionString,order=authParams;resolvedEnv;triggerMetadata,optional"`
	Username              string  `keda:"name=username,order=authParams;triggerMetadata,optional"`
	Password              string  `keda:"name=password,order=authParams;resolvedEnv;triggerMetadata,optional"`
	Host                  string  `keda:"name=host,order=authParams;triggerMetadata,optional"`
	Port                  int     `keda:"name=port,order=authParams;triggerMetadata,optional"`
	Database              string  `keda:"name=database,order=authParams;triggerMetadata,optional"`
	Query                 string  `keda:"name=query,order=triggerMetadata"`
	TargetValue           float64 `keda:"name=targetValue,order=triggerMetadata"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue,order=triggerMetadata,optional,default=0"`

	TriggerIndex int

	WorkloadIdentityResource      string `keda:"name=WorkloadIdentityResource,order=authParams;triggerMetadata,optional"`
	WorkloadIdentityClientID      string
	WorkloadIdentityTenantID      string
	WorkloadIdentityAuthorityHost string
}

func (m *mssqlMetadata) Validate() error {
	if m.ConnectionString == "" && m.Host == "" {
		return fmt.Errorf("must provide either connectionstring or host")
	}
	return nil
}

func NewMSSQLScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "mssql_scaler")

	meta, err := parseMSSQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing mssql metadata: %w", err)
	}

	conn, err := newMSSQLConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing mssql connection: %w", err)
	}

	return &mssqlScaler{
		metricType: metricType,
		metadata:   meta,
		connection: conn,
		logger:     logger,
	}, nil
}

func parseMSSQLMetadata(config *scalersconfig.ScalerConfig) (mssqlMetadata, error) {
	meta := mssqlMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, err
	}

	meta.TriggerIndex = config.TriggerIndex

	if config.PodIdentity.Provider == v1alpha1.PodIdentityProviderAzureWorkload {
		if config.AuthParams["workloadIdentityResource"] != "" {
			meta.WorkloadIdentityClientID = config.PodIdentity.GetIdentityID()
			meta.WorkloadIdentityTenantID = config.PodIdentity.GetIdentityTenantID()
			meta.WorkloadIdentityAuthorityHost = config.PodIdentity.GetIdentityAuthorityHost()
			meta.WorkloadIdentityResource = config.AuthParams["workloadIdentityResource"]
		}
	}

	return meta, nil
}

func newMSSQLConnection(meta mssqlMetadata, logger logr.Logger) (*sql.DB, error) {
	connStr := getMSSQLConnectionString(meta)

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error opening mssql: %s", err))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error pinging mssql: %s", err))
		return nil, err
	}

	return db, nil
}

func getMSSQLConnectionString(meta mssqlMetadata) string {
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
	var value float64

	// If using Azure Workload Identity, refresh the token
	if s.metadata.WorkloadIdentityResource != "" {
		if s.azureOAuth == nil {
			s.azureOAuth = azure.NewAzureADWorkloadIdentityTokenProvider(ctx, s.metadata.WorkloadIdentityClientID, s.metadata.WorkloadIdentityTenantID, s.metadata.WorkloadIdentityAuthorityHost, s.metadata.WorkloadIdentityResource)
		}

		err := s.azureOAuth.Refresh()
		if err != nil {
			return 0, fmt.Errorf("error refreshing Azure AD token: %w", err)
		}

		// Set the access token for the database connection
		err = s.connection.PingContext(ctx)
		if err != nil {
			return 0, fmt.Errorf("error pinging database: %w", err)
		}

		_, err = s.connection.ExecContext(ctx, "SET NOCOUNT ON; DECLARE @AccessToken NVARCHAR(MAX) = ?; EXEC sp_set_session_context @key=N'access_token', @value=@AccessToken;", s.azureOAuth.OAuthToken())
		if err != nil {
			return 0, fmt.Errorf("error setting access token: %w", err)
		}
	}

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
