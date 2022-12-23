package scalers

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type neo4jScaler struct {
	metricType v2.MetricTargetType
	metadata   *neo4jMetadata
	driver     neo4j.DriverWithContext
	logger     logr.Logger
}

type neo4jMetadata struct {
	// connectionString     string
	host                 string
	port                 string
	username             string
	password             string
	protocol             string
	query                string
	queryValue           float64
	activationQueryValue float64
	scalerIndex          int
}

func (s *neo4jScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, "neo4j"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *neo4jScaler) Close(ctx context.Context) error {
	if s.driver != nil {
		err := s.driver.Close(ctx)
		if err != nil {
			s.logger.Error(err, fmt.Sprintf("failed to close neo4j connection, because of %v", err))
			return err
		}
	}
	return nil
}

func (s *neo4jScaler) getQueryResult(ctx context.Context) (float64, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)
	result, err := session.ExecuteWrite(ctx, matchItemFn(s.metadata.query, ctx))
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Couldn't execute query string because of %v", err))
		return 0, err
	}
	res, err := strconv.ParseFloat((fmt.Sprintf("%v", result)), 64)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Couldn't parse to int because of %v", err))
		return 0, err
	}
	return res, nil
}

func matchItemFn(query string, ctx context.Context) neo4j.ManagedTransactionWork {
	return func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}
		if records.Peek(ctx) {
			record := records.Record()
			if err != nil {
				return nil, err
			}
			return record.Values[1], nil
		}

		return 0, nil
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
		queryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %v", val, err.Error())
		}
		meta.queryValue = queryValue
	} else {
		return nil, "", fmt.Errorf("no queryValue given")
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %v", val, err.Error())
		}
		meta.activationQueryValue = activationQueryValue
	}

	// switch {
	// case config.AuthParams["connectionString"] != "":
	// 	meta.connectionString = config.AuthParams["connectionString"]
	// case config.TriggerMetadata["connectionStringFromEnv"] != "":
	// 	meta.connectionString = config.ResolvedEnv[config.TriggerMetadata["connectionStringFromEnv"]]
	// default:
	// 	meta.connectionString = ""
	// 	host, err := GetFromAuthOrMeta(config, "host")
	// 	if err != nil {
	// 		return nil, "", err
	// 	}
	// 	meta.host = host

	// 	port, err := GetFromAuthOrMeta(config, "port")
	// 	if err != nil {
	// 		return nil, "", err
	// 	}
	// 	meta.port = port

	// 	username, err := GetFromAuthOrMeta(config, "username")
	// 	if err != nil {
	// 		return nil, "", err
	// 	}
	// 	meta.username = username

	// 	if config.AuthParams["password"] != "" {
	// 		meta.password = config.AuthParams["password"]
	// 	} else if config.TriggerMetadata["passwordFromEnv"] != "" {
	// 		meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
	// 	}
	// 	if len(meta.password) == 0 {
	// 		return nil, "", fmt.Errorf("no password given")
	// 	}
	// }

	// if meta.connectionString != "" {
	// 	connStr = meta.connectionString
	// } else {
	// Build connection str
	addr := net.JoinHostPort(meta.host, meta.port)
	// nosemgrep: db-connection-string
	connStr = meta.protocol + "://" + addr
	// }
	meta.scalerIndex = config.ScalerIndex
	return &meta, connStr, nil
}

// NewNeo4jScaler creates a new neo4j scaler instance
func NewNeo4jScaler(_ context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, connStr, err := parseNeo4jMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse neo4j metadata, because of %w", err)
	}
	fmt.Println("Metadata: ", meta)
	fmt.Println("Neo4j protocol: ", meta.protocol)
	fmt.Println("Neo4j connection string: ", connStr)
	driver, err := neo4j.NewDriverWithContext(connStr, neo4j.BasicAuth(meta.username, meta.password, ""))
	if err != nil {
		return nil, err
	}
	return &neo4jScaler{
		metricType: metricType,
		metadata:   meta,
		driver:     driver,
		logger:     InitializeLogger(config, "neo4j_scaler"),
	}, nil
}

// GetMetricsAndActivity query from neo4j,and return to external metrics and activity
func (s *neo4jScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	result, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect neo4j, because of %w", err)
	}

	metric := GenerateMetricInMili(metricName, result)

	return append([]external_metrics.ExternalMetricValue{}, metric), result > s.metadata.activationQueryValue, nil
}
