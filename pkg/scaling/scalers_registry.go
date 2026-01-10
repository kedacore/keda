/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaling

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// function type for building scalers
type scalerBuilder func(context.Context, client.Client, *scalersconfig.ScalerConfig) (scalers.Scaler, error)

// scalerBuilders holds all registered scaler builders (production and test)
var scalerBuilders = make(map[string]scalerBuilder)

// RegisterScalerBuilder registers a scaler builder for a given trigger type
func RegisterScalerBuilder(triggerType string, builder scalerBuilder) {
	scalerBuilders[triggerType] = builder
}

// Register all production scalers
func init() {
	RegisterScalerBuilder("activemq", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewActiveMQScaler(config)
	})

	RegisterScalerBuilder("apache-kafka", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewApacheKafkaScaler(ctx, config)
	})

	RegisterScalerBuilder("arangodb", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewArangoDBScaler(config)
	})

	RegisterScalerBuilder("artemis-queue", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewArtemisQueueScaler(config)
	})

	RegisterScalerBuilder("aws-cloudwatch", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAwsCloudwatchScaler(ctx, config)
	})

	RegisterScalerBuilder("aws-dynamodb", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAwsDynamoDBScaler(ctx, config)
	})

	RegisterScalerBuilder("aws-dynamodb-streams", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAwsDynamoDBStreamsScaler(ctx, config)
	})

	RegisterScalerBuilder("aws-kinesis-stream", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAwsKinesisStreamScaler(ctx, config)
	})

	RegisterScalerBuilder("aws-sqs-queue", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAwsSqsQueueScaler(ctx, config)
	})

	RegisterScalerBuilder("azure-app-insights", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureAppInsightsScaler(config)
	})

	RegisterScalerBuilder("azure-blob", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureBlobScaler(config)
	})

	RegisterScalerBuilder("azure-data-explorer", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureDataExplorerScaler(config)
	})

	RegisterScalerBuilder("azure-eventhub", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureEventHubScaler(config)
	})

	RegisterScalerBuilder("azure-log-analytics", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureLogAnalyticsScaler(config)
	})

	RegisterScalerBuilder("azure-monitor", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureMonitorScaler(config)
	})

	RegisterScalerBuilder("azure-pipelines", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzurePipelinesScaler(ctx, config)
	})

	RegisterScalerBuilder("azure-queue", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureQueueScaler(config)
	})

	RegisterScalerBuilder("azure-servicebus", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewAzureServiceBusScaler(ctx, config)
	})

	RegisterScalerBuilder("beanstalkd", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewBeanstalkdScaler(config)
	})

	RegisterScalerBuilder("cassandra", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewCassandraScaler(config)
	})

	RegisterScalerBuilder("couchdb", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewCouchDBScaler(ctx, config)
	})

	RegisterScalerBuilder("cpu", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewCPUMemoryScaler(corev1.ResourceCPU, config)
	})

	RegisterScalerBuilder("cron", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewCronScaler(config)
	})

	RegisterScalerBuilder("datadog", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewDatadogScaler(config)
	})

	RegisterScalerBuilder("dynatrace", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewDynatraceScaler(config)
	})

	RegisterScalerBuilder("elasticsearch", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewElasticsearchScaler(config)
	})

	RegisterScalerBuilder("etcd", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewEtcdScaler(config)
	})

	RegisterScalerBuilder("external", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewExternalScaler(config)
	})

	RegisterScalerBuilder("external-push", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewExternalPushScaler(config)
	})

	RegisterScalerBuilder("forgejo-runner", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewForgejoRunnerScaler(config)
	})

	RegisterScalerBuilder("gcp-cloudtasks", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewGcpCloudTasksScaler(config)
	})

	RegisterScalerBuilder("gcp-pubsub", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewPubSubScaler(config)
	})

	RegisterScalerBuilder("gcp-stackdriver", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewStackdriverScaler(ctx, config)
	})

	RegisterScalerBuilder("gcp-storage", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewGcsScaler(config)
	})

	RegisterScalerBuilder("github-runner", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewGitHubRunnerScaler(config)
	})

	RegisterScalerBuilder("graphite", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewGraphiteScaler(config)
	})

	RegisterScalerBuilder("huawei-cloudeye", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewHuaweiCloudeyeScaler(config)
	})

	RegisterScalerBuilder("ibmmq", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewIBMMQScaler(config)
	})

	RegisterScalerBuilder("influxdb", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewInfluxDBScaler(config)
	})

	RegisterScalerBuilder("kafka", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewKafkaScaler(ctx, config)
	})

	RegisterScalerBuilder("kubernetes-resource", func(_ context.Context, c client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewKubernetesResourceScaler(c, config)
	})

	RegisterScalerBuilder("kubernetes-workload", func(_ context.Context, c client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewKubernetesWorkloadScaler(c, config)
	})

	RegisterScalerBuilder("liiklus", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewLiiklusScaler(config)
	})

	RegisterScalerBuilder("loki", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewLokiScaler(config)
	})

	RegisterScalerBuilder("memory", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewCPUMemoryScaler(corev1.ResourceMemory, config)
	})

	RegisterScalerBuilder("metrics-api", func(_ context.Context, c client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewMetricsAPIScaler(config, c)
	})

	RegisterScalerBuilder("mongodb", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewMongoDBScaler(ctx, config)
	})

	RegisterScalerBuilder("mssql", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewMSSQLScaler(config)
	})

	RegisterScalerBuilder("mysql", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewMySQLScaler(config)
	})

	RegisterScalerBuilder("nats-jetstream", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewNATSJetStreamScaler(config)
	})

	RegisterScalerBuilder("new-relic", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewNewRelicScaler(config)
	})

	RegisterScalerBuilder("nsq", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewNSQScaler(config)
	})

	RegisterScalerBuilder("openstack-metric", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewOpenstackMetricScaler(ctx, config)
	})

	RegisterScalerBuilder("openstack-swift", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewOpenstackSwiftScaler(config)
	})

	RegisterScalerBuilder("postgresql", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewPostgreSQLScaler(ctx, config)
	})

	RegisterScalerBuilder("predictkube", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewPredictKubeScaler(ctx, config)
	})

	RegisterScalerBuilder("prometheus", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewPrometheusScaler(config)
	})

	RegisterScalerBuilder("pulsar", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewPulsarScaler(config)
	})

	RegisterScalerBuilder("rabbitmq", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRabbitMQScaler(config)
	})

	RegisterScalerBuilder("redis", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRedisScaler(ctx, false, false, config)
	})

	RegisterScalerBuilder("redis-cluster", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRedisScaler(ctx, true, false, config)
	})

	RegisterScalerBuilder("redis-cluster-streams", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRedisStreamsScaler(ctx, true, false, config)
	})

	RegisterScalerBuilder("redis-sentinel", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRedisScaler(ctx, false, true, config)
	})

	RegisterScalerBuilder("redis-sentinel-streams", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRedisStreamsScaler(ctx, false, true, config)
	})

	RegisterScalerBuilder("redis-streams", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewRedisStreamsScaler(ctx, false, false, config)
	})

	RegisterScalerBuilder("selenium-grid", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSeleniumGridScaler(config)
	})

	RegisterScalerBuilder("solace-direct-messaging", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSolaceDMScaler(config)
	})

	RegisterScalerBuilder("solace-event-queue", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSolaceScaler(config)
	})

	RegisterScalerBuilder("solarwinds", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSolarWindsScaler(config)
	})

	RegisterScalerBuilder("solr", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSolrScaler(config)
	})

	RegisterScalerBuilder("splunk", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSplunkScaler(config)
	})

	RegisterScalerBuilder("splunk-observability", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSplunkObservabilityScaler(config)
	})

	RegisterScalerBuilder("stan", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewStanScaler(config)
	})

	RegisterScalerBuilder("sumologic", func(_ context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewSumologicScaler(config)
	})

	RegisterScalerBuilder("temporal", func(ctx context.Context, _ client.Client, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		return scalers.NewTemporalScaler(ctx, config)
	})
}
