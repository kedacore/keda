package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"

	// PostreSQL drive required for this scaler
	_ "github.com/jackc/pgx/v5/stdlib"
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
	targetQueryValue           float64
	activationTargetQueryValue float64
	connection                 string
	query                      string
	triggerIndex               int
	azureAuthContext           azureAuthContext
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
	meta := postgreSQLMetadata{}

	authPodIdentity := kedav1alpha1.AuthPodIdentity{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, authPodIdentity, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["targetQueryValue"]; ok {
		targetQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, authPodIdentity, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.targetQueryValue = targetQueryValue
	} else {
		if config.AsMetricSource {
			meta.targetQueryValue = 0
		} else {
			return nil, authPodIdentity, fmt.Errorf("no targetQueryValue given")
		}
	}

	meta.activationTargetQueryValue = 0
	if val, ok := config.TriggerMetadata["activationTargetQueryValue"]; ok {
		activationTargetQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, authPodIdentity, fmt.Errorf("activationTargetQueryValue parsing error %w", err)
		}
		meta.activationTargetQueryValue = activationTargetQueryValue
	}

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		switch {
		case config.AuthParams["connection"] != "":
			meta.connection = config.AuthParams["connection"]
		case config.TriggerMetadata["connectionFromEnv"] != "":
			meta.connection = config.ResolvedEnv[config.TriggerMetadata["connectionFromEnv"]]
		default:
			params, err := buildConnArray(config)
			if err != nil {
				return nil, authPodIdentity, fmt.Errorf("failed to parse fields related to the connection")
			}

			var password string
			if config.AuthParams["password"] != "" {
				password = config.AuthParams["password"]
			} else if config.TriggerMetadata["passwordFromEnv"] != "" {
				password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
			}
			params = append(params, "password="+escapePostgreConnectionParameter(password))
			meta.connection = strings.Join(params, " ")
		}
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		params, err := buildConnArray(config)
		if err != nil {
			return nil, authPodIdentity, fmt.Errorf("failed to parse fields related to the connection")
		}

		cred, err := azure.NewChainedCredential(logger, config.PodIdentity.GetIdentityID(), config.PodIdentity.GetIdentityTenantID(), config.PodIdentity.Provider)
		if err != nil {
			return nil, authPodIdentity, err
		}
		meta.azureAuthContext.cred = cred

		params = append(params, "%PASSWORD%")
		meta.connection = strings.Join(params, " ")
	}
	meta.triggerIndex = config.TriggerIndex

	return &meta, authPodIdentity, nil
}

func buildConnArray(config *scalersconfig.ScalerConfig) ([]string, error) {
	var params []string

	host, err := GetFromAuthOrMeta(config, "host")
	if err != nil {
		return nil, err
	}

	port, err := GetFromAuthOrMeta(config, "port")
	if err != nil {
		return nil, err
	}

	userName, err := GetFromAuthOrMeta(config, "userName")
	if err != nil {
		return nil, err
	}

	dbName, err := GetFromAuthOrMeta(config, "dbName")
	if err != nil {
		return nil, err
	}

	sslmode, err := GetFromAuthOrMeta(config, "sslmode")
	if err != nil {
		return nil, err
	}
	params = append(params, "host="+escapePostgreConnectionParameter(host))
	params = append(params, "port="+escapePostgreConnectionParameter(port))
	params = append(params, "user="+escapePostgreConnectionParameter(userName))
	params = append(params, "dbname="+escapePostgreConnectionParameter(dbName))
	params = append(params, "sslmode="+escapePostgreConnectionParameter(sslmode))

	return params, nil
}

func getConnection(ctx context.Context, meta *postgreSQLMetadata, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger) (*sql.DB, error) {
	connectionString := meta.connection

	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		accessToken, err := getAzureAccessToken(ctx, meta, azureDatabasePostgresResource)
		if err != nil {
			return nil, err
		}
		newPasswordField := "password=" + escapePostgreConnectionParameter(accessToken)
		connectionString = passwordConnPattern.ReplaceAllString(meta.connection, newPasswordField)
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

	// Only one Azure case now but maybe more in the future.
	switch s.podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		if s.metadata.azureAuthContext.token.ExpiresOn.After(time.Now().Add(time.Second * 60)) {
			newConnection, err := getConnection(ctx, s.metadata, s.podIdentity, s.logger)
			if err != nil {
				return 0, fmt.Errorf("error establishing postgreSQL connection: %w", err)
			}
			s.connection = newConnection
		}
	}

	err := s.connection.QueryRowContext(ctx, s.metadata.query).Scan(&id)
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
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetQueryValue),
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

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationTargetQueryValue, nil
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
