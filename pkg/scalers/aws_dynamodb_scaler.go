package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type awsDynamoDBScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *awsDynamoDBMetadata
	dbClient   dynamodbiface.DynamoDBAPI
	logger     logr.Logger
}

type awsDynamoDBMetadata struct {
	tableName                 string
	awsRegion                 string
	keyConditionExpression    string
	expressionAttributeNames  map[string]*string
	expressionAttributeValues map[string]*dynamodb.AttributeValue
	targetValue               int64
	activationTargetValue     int64
	awsAuthorization          awsAuthorizationMetadata
	scalerIndex               int
	metricName                string
}

func NewAwsDynamoDBScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAwsDynamoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing DynamoDb metadata: %s", err)
	}

	return &awsDynamoDBScaler{
		metricType: metricType,
		metadata:   meta,
		dbClient:   createDynamoDBClient(meta),
		logger:     InitializeLogger(config, "aws_dynamodb_scaler"),
	}, nil
}

func parseAwsDynamoDBMetadata(config *ScalerConfig) (*awsDynamoDBMetadata, error) {
	meta := awsDynamoDBMetadata{}

	if val, ok := config.TriggerMetadata["tableName"]; ok && val != "" {
		meta.tableName = val
	} else {
		return nil, fmt.Errorf("no tableName given")
	}

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	if val, ok := config.TriggerMetadata["keyConditionExpression"]; ok && val != "" {
		meta.keyConditionExpression = val
	} else {
		return nil, fmt.Errorf("no keyConditionExpression given")
	}

	if val, ok := config.TriggerMetadata["expressionAttributeNames"]; ok && val != "" {
		names, err := json2Map(val)

		if err != nil {
			return nil, fmt.Errorf("error parsing expressionAttributeNames: %s", err)
		}

		meta.expressionAttributeNames = names
	} else {
		return nil, fmt.Errorf("no expressionAttributeNames given")
	}

	if val, ok := config.TriggerMetadata["expressionAttributeValues"]; ok && val != "" {
		values, err := json2DynamoMap(val)

		if err != nil {
			return nil, fmt.Errorf("error parsing expressionAttributeValues: %s", err)
		}

		meta.expressionAttributeValues = values
	} else {
		return nil, fmt.Errorf("no expressionAttributeValues given")
	}

	if val, ok := config.TriggerMetadata["targetValue"]; ok && val != "" {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata targetValue")
		}

		meta.targetValue = n
	} else {
		return nil, fmt.Errorf("no targetValue given")
	}

	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok && val != "" {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata targetValue")
		}

		meta.activationTargetValue = n
	} else {
		meta.activationTargetValue = 0
	}

	auth, err := getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.scalerIndex = config.ScalerIndex

	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex,
		kedautil.NormalizeString(fmt.Sprintf("aws-dynamodb-%s", meta.tableName)))

	return &meta, nil
}

func createDynamoDBClient(meta *awsDynamoDBMetadata) *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(meta.awsRegion),
	}))

	var dbClient *dynamodb.DynamoDB

	if !meta.awsAuthorization.podIdentityOwner {
		dbClient = dynamodb.New(sess, &aws.Config{
			Region: aws.String(meta.awsRegion),
		})

		return dbClient
	}

	creds := credentials.NewStaticCredentials(meta.awsAuthorization.awsAccessKeyID, meta.awsAuthorization.awsSecretAccessKey, "")

	if meta.awsAuthorization.awsRoleArn != "" {
		creds = stscreds.NewCredentials(sess, meta.awsAuthorization.awsRoleArn)
	}

	dbClient = dynamodb.New(sess, &aws.Config{
		Region:      aws.String(meta.awsRegion),
		Credentials: creds,
	})

	return dbClient
}

func (s *awsDynamoDBScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	metricValue, err := s.GetQueryMetrics()
	if err != nil {
		s.logger.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, metricValue)

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *awsDynamoDBScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}

	return []v2beta2.MetricSpec{
		metricSpec,
	}
}

func (s *awsDynamoDBScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.GetQueryMetrics()
	if err != nil {
		return false, fmt.Errorf("error inspecting aws-dynamodb: %s", err)
	}

	return messages > float64(s.metadata.activationTargetValue), nil
}

func (s *awsDynamoDBScaler) Close(context.Context) error {
	return nil
}

func (s *awsDynamoDBScaler) GetQueryMetrics() (float64, error) {
	dimensions := dynamodb.QueryInput{
		TableName:                 aws.String(s.metadata.tableName),
		KeyConditionExpression:    aws.String(s.metadata.keyConditionExpression),
		ExpressionAttributeNames:  s.metadata.expressionAttributeNames,
		ExpressionAttributeValues: s.metadata.expressionAttributeValues,
	}

	res, err := s.dbClient.Query(&dimensions)
	if err != nil {
		s.logger.Error(err, "Failed to get output")
		return 0, err
	}

	return float64(*res.Count), nil
}

// json2Map convert Json to map[string]string
func json2Map(js string) (m map[string]*string, err error) {
	err = bson.UnmarshalExtJSON([]byte(js), true, &m)
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, errors.New("empty map")
	}
	return m, err
}

// json2DynamoMap converts Json to map[string]*dynamoDb.AttributeValue
func json2DynamoMap(js string) (m map[string]*dynamodb.AttributeValue, err error) {
	err = json.Unmarshal([]byte(js), &m)

	if err != nil {
		return nil, err
	}
	return m, err
}
