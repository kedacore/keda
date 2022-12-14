package scalers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type neo4jScaler struct {
	metricType v2.MetricTargetType
	metadata   *neo4jMetadata
	driver     neo4j.DriverWithContext
	logger     logr.Logger
}

type neo4jMetadata struct {
	connectionString     string
	host                 string
	port                 string
	username             string
	password             string
	query                string
	queryValue           int64
	activationQueryValue int64
	metricName           string
	scalerIndex          int
}

func (n *neo4jScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(n.metadata.scalerIndex, n.metadata.metricName),
		},
		Target: GetMetricTarget(n.metricType, n.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (n *neo4jScaler) Close(ctx context.Context) error {
	if n.driver != nil {
		err := n.driver.Close(ctx)
		if err != nil {
			n.logger.Error(err, fmt.Sprintf("failed to close neo4j connection, because of %v", err))
			return err
		}
	}
	return nil
}

func (n *neo4jScaler) getQueryResult(ctx context.Context) (int64, error) {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)
	result, err := session.ExecuteWrite(ctx, matchItemFn(ctx, n.metadata.query))
	if err != nil {
		n.logger.Error(err, fmt.Sprintf("Couldn't execute query string because of %v", err))
		return 0, err
	}
	fmt.Println("Result of query: ", result)
	res, err := strconv.ParseInt((fmt.Sprintf("%v", result)), 10, 64)
	if err != nil {
		n.logger.Error(err, fmt.Sprintf("Couldn't parse to int because of %v", err))
		return 0, err
	}
	return res, nil
}

func matchItemFn(ctx context.Context, queries string) neo4j.ManagedTransactionWork {
	return func(tx neo4j.ManagedTransaction) (any, error) {
		query := strings.Split(queries, ";")
		_, err := tx.Run(ctx, query[0], nil)
		if err != nil {
			return nil, err
		}
		records, err := tx.Run(ctx, query[1], nil)
		if err != nil {
			return nil, err
		}
		record, err := records.Single(ctx)
		if err != nil {
			return nil, err
		}
		return record.Values[1], nil
	}
}

func parseNeo4jMetadata(config *ScalerConfig) (*neo4jMetadata, string, error) {
	var connStr string
	meta := neo4jMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, "", fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %v", val, err.Error())
		}
		meta.queryValue = queryValue
	} else {
		return nil, "", fmt.Errorf("no queryValue given")
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %v", val, err.Error())
		}
		meta.activationQueryValue = activationQueryValue
	}

	switch {
	case config.AuthParams["connectionString"] != "":
		meta.connectionString = config.AuthParams["connectionString"]
	case config.TriggerMetadata["connectionStringFromEnv"] != "":
		meta.connectionString = config.ResolvedEnv[config.TriggerMetadata["connectionStringFromEnv"]]
	default:
		meta.connectionString = ""
		host, err := GetFromAuthOrMeta(config, "host")
		if err != nil {
			return nil, "", err
		}
		meta.host = host

		port, err := GetFromAuthOrMeta(config, "port")
		if err != nil {
			return nil, "", err
		}
		meta.port = port

		username, err := GetFromAuthOrMeta(config, "username")
		if err != nil {
			return nil, "", err
		}
		meta.username = username

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}
		if len(meta.password) == 0 {
			return nil, "", fmt.Errorf("no password given")
		}
	}

	if meta.connectionString != "" {
		connStr = meta.connectionString
	} else {
		// Build connection str
		addr := net.JoinHostPort(meta.host, meta.port)
		connStr = "neo4j://" + addr
	}
	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("neo4j-%s", val))
	}
	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString("neo4j"))
	meta.scalerIndex = config.ScalerIndex
	return &meta, connStr, nil
}

func NewNeo4jScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, connStr, err := parseNeo4jMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parsing neo4j metadata, because of %v", err)
	}

	driver, err := neo4j.NewDriverWithContext(connStr, neo4j.BasicAuth(meta.username, meta.password, ""))
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	fmt.Println("Connected to neo4j")

	return &neo4jScaler{
		metricType: metricType,
		metadata:   meta,
		driver:     driver,
		logger:     InitializeLogger(config, "neo4j_scaler"),
	}, nil
}

// GetMetricsAndActivity query from neo4j,and return to external metrics and activity
func (n *neo4jScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	result, err := n.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect neo4j, because of %v", err)
	}

	metric := GenerateMetricInMili(metricName, float64(result))

	return append([]external_metrics.ExternalMetricValue{}, metric), result > n.metadata.activationQueryValue, nil
}
