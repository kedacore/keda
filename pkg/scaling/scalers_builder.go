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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

/// --------------------------------------------------------------------------- ///
/// ----------            Scaler-Building related methods             --------- ///
/// --------------------------------------------------------------------------- ///

// buildScalers returns list of Scalers for the specified triggers
func (h *scaleHandler) buildScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, podTemplateSpec *corev1.PodTemplateSpec, containerName string, asMetricSource bool) ([]cache.ScalerBuilder, error) {
	logger := log.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	var err error
	resolvedEnv := make(map[string]string)
	result := make([]cache.ScalerBuilder, 0, len(withTriggers.Spec.Triggers))

	for i, t := range withTriggers.Spec.Triggers {
		triggerIndex, trigger := i, t

		factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
			if podTemplateSpec != nil {
				resolvedEnv, err = resolver.ResolveContainerEnv(ctx, h.client, logger, &podTemplateSpec.Spec, containerName, withTriggers.Namespace, h.secretsLister)
				if err != nil {
					return nil, nil, fmt.Errorf("error resolving secrets for ScaleTarget: %w", err)
				}
			}
			config := &scalersconfig.ScalerConfig{
				ScalableObjectName:      withTriggers.Name,
				ScalableObjectNamespace: withTriggers.Namespace,
				ScalableObjectType:      withTriggers.Kind,
				TriggerName:             trigger.Name,
				TriggerMetadata:         trigger.Metadata,
				TriggerType:             trigger.Type,
				TriggerUseCachedMetrics: trigger.UseCachedMetrics,
				ResolvedEnv:             resolvedEnv,
				AuthParams:              make(map[string]string),
				GlobalHTTPTimeout:       h.globalHTTPTimeout,
				TriggerIndex:            triggerIndex,
				MetricType:              trigger.MetricType,
				AsMetricSource:          asMetricSource,
				ScaledObject:            withTriggers,
				Recorder:                h.recorder,
				TriggerUniqueKey:        fmt.Sprintf("%s-%s-%s-%d", withTriggers.Kind, withTriggers.Namespace, withTriggers.Name, triggerIndex),
			}

			authParams, podIdentity, err := resolver.ResolveAuthRefAndPodIdentity(ctx, h.client, logger, trigger.AuthenticationRef, podTemplateSpec, withTriggers.Namespace, h.secretsLister)
			switch podIdentity.Provider {
			case kedav1alpha1.PodIdentityProviderAwsEKS:
				// FIXME: Delete this for v3
				logger.Info("WARNING: AWS EKS Identity has been deprecated in favor of AWS Identity and will be removed from KEDA on v3")
			default:
			}

			if err != nil {
				return nil, nil, err
			}
			config.AuthParams = authParams
			config.PodIdentity = podIdentity
			scaler, err := buildScaler(ctx, h.client, trigger.Type, config)
			return scaler, config, err
		}

		// nosemgrep: invalid-usage-of-modified-variable
		scaler, config, err := factory()
		if err != nil {
			h.recorder.Event(withTriggers, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			logger.Error(err, "error resolving auth params", "triggerIndex", triggerIndex)
			if scaler != nil {
				_ = scaler.Close(ctx)
			}
			for _, builder := range result {
				_ = builder.Scaler.Close(ctx)
			}
			return nil, err
		}
		msg := fmt.Sprintf(message.ScalerIsBuiltMsg, trigger.Type)
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStarted, msg)

		result = append(result, cache.ScalerBuilder{
			Scaler:       scaler,
			ScalerConfig: *config,
			Factory:      factory,
		})
	}

	return result, nil
}

// buildScaler builds a scaler form input config and trigger type
func buildScaler(ctx context.Context, client client.Client, triggerType string, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
	// TRIGGERS-START
	switch triggerType {
	case "activemq":
		return scalers.NewActiveMQScaler(config)
	case "apache-kafka":
		return scalers.NewApacheKafkaScaler(ctx, config)
	case "arangodb":
		return scalers.NewArangoDBScaler(config)
	case "artemis-queue":
		return scalers.NewArtemisQueueScaler(config)
	case "aws-cloudwatch":
		return scalers.NewAwsCloudwatchScaler(ctx, config)
	case "aws-dynamodb":
		return scalers.NewAwsDynamoDBScaler(ctx, config)
	case "aws-dynamodb-streams":
		return scalers.NewAwsDynamoDBStreamsScaler(ctx, config)
	case "aws-kinesis-stream":
		return scalers.NewAwsKinesisStreamScaler(ctx, config)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(ctx, config)
	case "azure-app-insights":
		return scalers.NewAzureAppInsightsScaler(config)
	case "azure-blob":
		return scalers.NewAzureBlobScaler(config)
	case "azure-data-explorer":
		return scalers.NewAzureDataExplorerScaler(config)
	case "azure-eventhub":
		return scalers.NewAzureEventHubScaler(config)
	case "azure-log-analytics":
		return scalers.NewAzureLogAnalyticsScaler(config)
	case "azure-monitor":
		return scalers.NewAzureMonitorScaler(config)
	case "azure-pipelines":
		return scalers.NewAzurePipelinesScaler(ctx, config)
	case "azure-queue":
		return scalers.NewAzureQueueScaler(config)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(ctx, config)
	case "beanstalkd":
		return scalers.NewBeanstalkdScaler(config)
	case "cassandra":
		return scalers.NewCassandraScaler(config)
	case "couchdb":
		return scalers.NewCouchDBScaler(ctx, config)
	case "cpu":
		return scalers.NewCPUMemoryScaler(corev1.ResourceCPU, config)
	case "cron":
		return scalers.NewCronScaler(config)
	case "datadog":
		return scalers.NewDatadogScaler(ctx, config)
	case "dynatrace":
		return scalers.NewDynatraceScaler(config)
	case "elasticsearch":
		return scalers.NewElasticsearchScaler(config)
	case "etcd":
		return scalers.NewEtcdScaler(config)
	case "external":
		return scalers.NewExternalScaler(config)
	// TODO: use other way for test.
	case "external-mock":
		return scalers.NewExternalMockScaler(config)
	case "external-push":
		return scalers.NewExternalPushScaler(config)
	case "forgejo-runner":
		return scalers.NewForgejoRunnerScaler(config)
	case "gcp-cloudtasks":
		return scalers.NewGcpCloudTasksScaler(config)
	case "gcp-pubsub":
		return scalers.NewPubSubScaler(config)
	case "gcp-stackdriver":
		return scalers.NewStackdriverScaler(ctx, config)
	case "gcp-storage":
		return scalers.NewGcsScaler(config)
	case "github-runner":
		return scalers.NewGitHubRunnerScaler(config)
	case "graphite":
		return scalers.NewGraphiteScaler(config)
	case "huawei-cloudeye":
		return scalers.NewHuaweiCloudeyeScaler(config)
	case "ibmmq":
		return scalers.NewIBMMQScaler(config)
	case "influxdb":
		return scalers.NewInfluxDBScaler(config)
	case "kafka":
		return scalers.NewKafkaScaler(ctx, config)
	case "kubernetes-workload":
		return scalers.NewKubernetesWorkloadScaler(client, config)
	case "liiklus":
		return scalers.NewLiiklusScaler(config)
	case "loki":
		return scalers.NewLokiScaler(config)
	case "memory":
		return scalers.NewCPUMemoryScaler(corev1.ResourceMemory, config)
	case "metrics-api":
		return scalers.NewMetricsAPIScaler(config)
	case "mongodb":
		return scalers.NewMongoDBScaler(ctx, config)
	case "mssql":
		return scalers.NewMSSQLScaler(config)
	case "mysql":
		return scalers.NewMySQLScaler(config)
	case "nats-jetstream":
		return scalers.NewNATSJetStreamScaler(config)
	case "new-relic":
		return scalers.NewNewRelicScaler(config)
	case "nsq":
		return scalers.NewNSQScaler(config)
	case "openstack-metric":
		return scalers.NewOpenstackMetricScaler(ctx, config)
	case "openstack-swift":
		return scalers.NewOpenstackSwiftScaler(config)
	case "postgresql":
		return scalers.NewPostgreSQLScaler(ctx, config)
	case "predictkube":
		return scalers.NewPredictKubeScaler(ctx, config)
	case "prometheus":
		return scalers.NewPrometheusScaler(config)
	case "pulsar":
		return scalers.NewPulsarScaler(config)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(config)
	case "redis":
		return scalers.NewRedisScaler(ctx, false, false, config)
	case "redis-cluster":
		return scalers.NewRedisScaler(ctx, true, false, config)
	case "redis-cluster-streams":
		return scalers.NewRedisStreamsScaler(ctx, true, false, config)
	case "redis-sentinel":
		return scalers.NewRedisScaler(ctx, false, true, config)
	case "redis-sentinel-streams":
		return scalers.NewRedisStreamsScaler(ctx, false, true, config)
	case "redis-streams":
		return scalers.NewRedisStreamsScaler(ctx, false, false, config)
	case "selenium-grid":
		return scalers.NewSeleniumGridScaler(config)
	case "solace-event-queue":
		return scalers.NewSolaceScaler(config)
	case "solr":
		return scalers.NewSolrScaler(config)
	case "splunk":
		return scalers.NewSplunkScaler(config)
	case "stan":
		return scalers.NewStanScaler(config)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", triggerType)
	}
	// TRIGGERS-END
}
