# Changelog

<!--
    New changelog entries must be inline with our changelog guidelines.
    Please refer to https://github.com/kedacore/keda/blob/main/CONTRIBUTING.md#Changelog to learn more.
-->

This changelog keeps track of work items that have been completed and are ready to be shipped in the next release.

To learn more about our roadmap, we recommend reading [this document](ROADMAP.md).

## Deprecations

To learn more about active deprecations, we recommend checking [GitHub Discussions](https://github.com/kedacore/keda/discussions/categories/deprecations).

## History

- [Unreleased](#unreleased)
- [v2.13.1](#v2131)
- [v2.13.0](#v2130)
- [v2.12.1](#v2121)
- [v2.12.0](#v2120)
- [v2.11.2](#v2112)
- [v2.11.1](#v2111)
- [v2.11.0](#v2110)
- [v2.10.1](#v2101)
- [v2.10.0](#v2100)
- [v2.9.3](#v293)
- [v2.9.2](#v292)
- [v2.9.1](#v291)
- [v2.9.0](#v290)
- [v2.8.1](#v281)
- [v2.8.0](#v280)
- [v2.7.1](#v271)
- [v2.7.0](#v270)
- [v2.6.1](#v261)
- [v2.6.0](#v260)
- [v2.5.0](#v250)
- [v2.4.0](#v240)
- [v2.3.0](#v230)
- [v2.2.0](#v220)
- [v2.1.0](#v210)
- [v2.0.0](#v200)
- [v1.5.0](#v150)
- [v1.4.1](#v141)
- [v1.4.0](#v140)
- [v1.3.0](#v130)
- [v1.2.0](#v120)
- [v1.1.0](#v110)
- [v1.0.0](#v100)

## Unreleased

### New

- **General**: Provide capability to filter CloudEvents ([#3533](https://github.com/kedacore/keda/issues/3533))
- **NATS Scaler**: Add TLS authentication ([#2296](https://github.com/kedacore/keda/issues/2296))
- **ScaledObject**: Ability to specify `initialCooldownPeriod` ([#5008](https://github.com/kedacore/keda/issues/5008))



#### Experimental

Here is an overview of all new **experimental** features:

- **General**: Introduce Azure Event Grid as a new CloudEvent destination ([#3587](https://github.com/kedacore/keda/issues/3587))

### Improvements

- **General**: Add active trigger name in ScaledObject's scale out event ([#5577](https://github.com/kedacore/keda/issues/5577))
- **General**: Add command-line flag in Adapter to allow override of gRPC Authority Header ([#5449](https://github.com/kedacore/keda/issues/5449))
- **General**: Add GRPC Client and Server metrics ([#5502](https://github.com/kedacore/keda/issues/5502))
- **General**: Add OPENTELEMETRY flag in e2e test YAML ([#5375](https://github.com/kedacore/keda/issues/5375))
- **General**: Add support for cross tenant/cloud authentication when using Azure Workload Identity for TriggerAuthentication ([#5441](https://github.com/kedacore/keda/issues/5441))
- **General**: Add `validations.keda.sh/hpa-ownership` annotation to HPA to disable ownership validation ([#5516](https://github.com/kedacore/keda/issues/5516))
- **General**: Improve Prometheus metrics to align with best practices ([#4854](https://github.com/kedacore/keda/issues/4854))
- **General**: Support csv-format for WATCH_NAMESPACE env var ([#5670](https://github.com/kedacore/keda/issues/5670))
- **Azure Event Hub Scaler**: Remove usage of checkpoint offsets to account for SDK checkpointing implementation changes ([#5574](https://github.com/kedacore/keda/issues/5574))
- **GCP Pub/Sub Scaler**: Add support for resolving resource names from the scale target's environment ([#5693](https://github.com/kedacore/keda/issues/5693))
- **GCP Stackdriver Scaler**: Add missing parameters 'rate' and 'count' for GCP Stackdriver Scaler alignment ([#5633](https://github.com/kedacore/keda/issues/5633))
- **Metrics API Scaler**: Add support for various formats: json, xml, yaml, prometheus ([#2633](https://github.com/kedacore/keda/issues/2633))
- **MongoDB Scaler**: Add scheme field support srv record ([#5544](https://github.com/kedacore/keda/issues/5544))

### Fixes

- **General**: Fix CVE-2024-28180 in github.com/go-jose/go-jose/v3 ([#5617](https://github.com/kedacore/keda/pull/5617))
- **General**: Log field `ScaledJob` no longer have conflicting types ([#5592](https://github.com/kedacore/keda/pull/5592))
- **General**: Prometheus metrics shows errors correctly ([#5597](https://github.com/kedacore/keda/issues/5597)|[#5663](https://github.com/kedacore/keda/issues/5663))
- **General**: Validate empty array value of triggers in ScaledObject/ScaledJob creation ([#5520](https://github.com/kedacore/keda/issues/5520))
- **GitHub Runner Scaler**: Fixed `in_progress` detection on running jobs instead of just `queued` ([#5604](https://github.com/kedacore/keda/issues/5604))
- **New Relic Scaler**: Consider empty results set from query executer ([#5619](https://github.com/kedacore/keda/pull/5619))
- **RabbitMQ Scaler**: HTTP Connections respect TLS configuration ([#5668](https://github.com/kedacore/keda/issues/5668))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- Various Prometheus metrics have been renamed to follow the preferred naming conventions. The old ones are still available, but will be removed in the future ([#4854](https://github.com/kedacore/keda/issues/4854)).

### Breaking Changes

- **General**: TODO ([#XXX](https://github.com/kedacore/keda/issues/XXX))

### Other

- **General**: Allow E2E tests to be run against existing KEDA and/or Kafka installation ([#5595](https://github.com/kedacore/keda/pull/5595))
- **General**: Improve readability of utility function getParameterFromConfigV2 ([#5037](https://github.com/kedacore/keda/issues/5037))
- **General**: Introduce ENABLE_OPENTELEMETRY in deploying/testing process  ([#5375](https://github.com/kedacore/keda/issues/5375)|[#5578](https://github.com/kedacore/keda/issues/5578))
- **General**: Migrate away from unmaintained golang/mock and use uber/gomock ([#5440](https://github.com/kedacore/keda/issues/5440))
- **General**: Minor refactor to reduce copy/paste code in ScaledObject webhook ([#5397](https://github.com/kedacore/keda/issues/5397))
- **General**: No need to list all secret in the namespace to find just one ([#5669](https://github.com/kedacore/keda/pull/5669))
- **Kafka**: Expose GSSAPI service name  ([#5474](https://github.com/kedacore/keda/issues/5474))

## v2.13.1

### Fixes

- **General**: Fix release asset should specify the version in `keda-*-core.yaml`([#5484](https://github.com/kedacore/keda/issues/5484))
- **GCP Scalers**: Properly close the connection during the scaler cleaning process ([#5448](https://github.com/kedacore/keda/issues/5448))
- **GCP Scalers**: Restore previous time horizon to prevent querying issues ([#5429](https://github.com/kedacore/keda/issues/5429))
- **Prometheus Scaler**: Fix for missing AWS region from metadata ([#5419](https://github.com/kedacore/keda/issues/5419))

## v2.13.0

### New

- **General**: Adds support for GCP Secret Manager as a source for TriggerAuthentication ([#4831](https://github.com/kedacore/keda/issues/4831))
- **General**: Introduce new AWS Authentication ([#4134](https://github.com/kedacore/keda/issues/4134))
- **General**: Support TriggerAuthentication properties from ConfigMap ([#4830](https://github.com/kedacore/keda/issues/4830))
- **Azure Blob Storage Scaler**: Allow to authenticate to Azure Storage using SAS tokens ([#5393](https://github.com/kedacore/keda/issues/5393))
- **Azure Pipelines Scaler**: Add support for workload identity authentication ([#5013](https://github.com/kedacore/keda/issues/5013))
- **Azure Storage Queue Scaler**: Allow to authenticate to Azure Storage using SAS tokens ([#5393](https://github.com/kedacore/keda/issues/5393))
- **Kafka Scaler**: Add support for Kerberos authentication (SASL / GSSAPI) ([#4836](https://github.com/kedacore/keda/issues/4836))
- **Prometheus Metrics**: Expose prometheus metrics for ScaledJob resources ([#4798](https://github.com/kedacore/keda/issues/4798))
- **Prometheus Metrics**: Introduce paused ScaledObjects in Prometheus metrics ([#4430](https://github.com/kedacore/keda/issues/4430))
- **Prometheus Scaler**: Provide scaler for Amazon managed service for Prometheus ([#2214](https://github.com/kedacore/keda/issues/2214))

#### Experimental

Here is an overview of all new **experimental** features:

- **General**:  Emit CloudEvents on major KEDA events ([#3533](https://github.com/kedacore/keda/issues/3533)|[#5278](https://github.com/kedacore/keda/issues/5278))

### Improvements

- **General**: Add CloudEventSource metrics in Prometheus & OpenTelemetry ([#3531](https://github.com/kedacore/keda/issues/3531))
- **General**: Add RBAC permissions for list & watch LimitRange, and check default limits from LimitRange for validations ([#5377](https://github.com/kedacore/keda/pull/5377))
- **General**: Add validations for replica counts when creating ScaledObjects ([#5288](https://github.com/kedacore/keda/issues/5288))
- **General**: Bubble up AuthRef TriggerAuthentication errors as ScaledObject events ([#5190](https://github.com/kedacore/keda/issues/5190))
- **General**: Enhance pod identity role assumption in AWS by directly integrating with OIDC/Federation ([#5178](https://github.com/kedacore/keda/issues/5178))
- **General**: Fix issue where paused annotation being set to false still leads to ScaledObjects/ScaledJobs being paused ([#5215](https://github.com/kedacore/keda/issues/5215))
- **General**: Implement credentials cache for AWS Roles to reduce AWS API calls ([#5297](https://github.com/kedacore/keda/issues/5297))
- **General**: Request all ScaledObject/ScaledJob triggers in parallel ([#5276](https://github.com/kedacore/keda/issues/5276))
- **General**: Use client-side round-robin load balancing for gRPC calls ([#5224](https://github.com/kedacore/keda/issues/5224))
- **GCP PubSub Scaler**: Support distribution-valued metrics and metrics from topics ([#5070](https://github.com/kedacore/keda/issues/5070))
- **GCP Stackdriver Scaler**: Support valueIfNull parameter ([#5345](https://github.com/kedacore/keda/pull/5345))
- **Hashicorp Vault**: Add support to get secret that needs write operation (eg. `pki`) ([#5067](https://github.com/kedacore/keda/issues/5067))
- **Hashicorp Vault**: Fix operator panic when `spec.hashiCorpVault.credential.serviceAccount` is not set ([#4964](https://github.com/kedacore/keda/issues/4964))
- **Hashicorp Vault**: Fix operator panic when using root token to authenticate to vault server ([#5192](https://github.com/kedacore/keda/issues/5192))
- **Kafka Scaler**: Ability to set upper bound to the number of partitions with lag ([#3997](https://github.com/kedacore/keda/issues/3997))
- **Kafka Scaler**: Improve logging for Sarama client ([#5102](https://github.com/kedacore/keda/issues/5102))
- **Prometheus Scaler**: Add `queryParameters` parameter ([#4962](https://github.com/kedacore/keda/issues/4962))
- **Pulsar Scaler**: Support `endpointParams`` in Pulsar OAuth ([#5069](https://github.com/kedacore/keda/issues/5069))

### Fixes

- **General**: Admission webhook does not reject workloads with only resource limits provided ([#4802](https://github.com/kedacore/keda/issues/4802))
- **General**: Fix CVE-2023-39325 in golang.org/x/net ([#5122](https://github.com/kedacore/keda/issues/5122))
- **General**: Fix otelgrpc DoS vulnerability ([#5208](https://github.com/kedacore/keda/issues/5208))
- **General**: Fix Pod identity not being considered when scaled target is a CRD ([#5021](https://github.com/kedacore/keda/issues/5021))
- **General**: Prevented memory leak generated by not correctly cleaning http connections ([#5248](https://github.com/kedacore/keda/issues/5248))
- **General**: Prevented stuck status due to timeouts during scalers generation ([#5083](https://github.com/kedacore/keda/issues/5083))
- **General**: ScaledObject Validating Webhook should support `dry-run=server` requests ([#5306](https://github.com/kedacore/keda/issues/5306))
- **General**: Set `LeaderElectionNamespace` to PodNamespace so leader election works in OutOfCluster mode ([#5404](https://github.com/kedacore/keda/issues/5404))
- **AWS Scalers**: Ensure session tokens are included when instantiating AWS credentials ([#5156](https://github.com/kedacore/keda/issues/5156))
- **Azure Event Hub Scaler**: Improve unprocessedEventThreshold calculation ([#4250](https://github.com/kedacore/keda/issues/4250))
- **Azure Pipelines**: Prevent HTTP 400 errors due to `poolName` with spaces ([#5107](https://github.com/kedacore/keda/issues/5107))
- **GCP PubSub Scaler**: Added `project_id` to filter for metrics queries ([#5256](https://github.com/kedacore/keda/issues/5256))
- **GCP PubSub Scaler**: Respect default value of `value` ([#5093](https://github.com/kedacore/keda/issues/5093))
- **Github Runner Scaler**: Support for custom API endpoint ([#5387](https://github.com/kedacore/keda/issues/5387))
- **NATS JetSteam Scaler**: Raise an error if leader not found ([#5358](https://github.com/kedacore/keda/pull/5358))
- **Pulsar Scaler**: Fix panic when auth is not used ([#5271](https://github.com/kedacore/keda/issues/5271))
- **ScaledJobs**: Copy ScaledJob annotations to child Jobs ([#4594](https://github.com/kedacore/keda/issues/4594))

### Deprecations


You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- Remove support for Azure AD Pod Identity-based authentication ([#5035](https://github.com/kedacore/keda/issues/5035))

### Breaking Changes

- **General**: Clean up previously deprecated code in Azure Data Explorer Scaler about clientSecret for 2.13 release ([#5051](https://github.com/kedacore/keda/issues/5051))

### Other

- **General**: Bump K8s deps to 0.28.5 ([#5346](https://github.com/kedacore/keda/pull/5346))
- **General**: Fix CVE-2023-45142 in OpenTelemetry ([#5089](https://github.com/kedacore/keda/issues/5089))
- **General**: Fix logger in OpenTelemetry collector ([#5094](https://github.com/kedacore/keda/issues/5094))
- **General**: Fix lost commit from the newly created utility function ([#5037](https://github.com/kedacore/keda/issues/5037))
- **General**: Improve docker image build time through caches ([#5316](https://github.com/kedacore/keda/issues/5316))
- **General**: Reduce amount of gauge creations for OpenTelemetry metrics ([#5101](https://github.com/kedacore/keda/issues/5101))
- **General**: Refactor `scalers` package ([#5379](https://github.com/kedacore/keda/issues/5379))
- **General**: Removed not required RBAC permissions ([#5261](https://github.com/kedacore/keda/issues/5261))
- **General**: Support profiling for KEDA components ([#4789](https://github.com/kedacore/keda/issues/4789))
- **CPU scaler**: Wait for metrics window during CPU scaler tests ([#5294](https://github.com/kedacore/keda/pull/5294))
- **Hashicorp Vault**: Improve test coverage in `pkg/scaling/resolver/hashicorpvault_handler`  ([#5195](https://github.com/kedacore/keda/issues/5195))
- **Kafka Scaler**: Add more test cases for large value of LagThreshold ([#5354](https://github.com/kedacore/keda/issues/5354))
- **Openstack Scaler**: Use Gophercloud SDK ([#3439](https://github.com/kedacore/keda/issues/3439))

## v2.12.1

### Fixes

- **General**: Fix CVE-2023-39325 in golang.org/x/net ([#5122](https://github.com/kedacore/keda/issues/5122))
- **General**: Fix CVE-2023-45142 in Opentelemetry ([#5089](https://github.com/kedacore/keda/issues/5089))
- **General**: Fix logger in Opentelemetry collector ([#5094](https://github.com/kedacore/keda/issues/5094))
- **General**: Fix otelgrpc DoS vulnerability ([#5208](https://github.com/kedacore/keda/issues/5208))
- **General**: Prevented stuck status due to timeouts during scalers generation ([#5083](https://github.com/kedacore/keda/issues/5083))
- **Azure Pipelines**: No more HTTP 400 errors produced by poolName with spaces ([#5107](https://github.com/kedacore/keda/issues/5107))

## v2.12.0

### New

- **General**: Introduce new Google Cloud Tasks scaler ([#3613](https://github.com/kedacore/keda/issues/3613))
- **AWS SQS Scaler**: Support for scaling to include delayed messages. ([#4377](https://github.com/kedacore/keda/issues/4377))
- **Governance**: KEDA transitioned to CNCF Graduated project ([#63](https://github.com/kedacore/governance/issues/63))

#### Experimental

Here is an overview of all new **experimental** features:

- **General**: Introduce pushing operational metrics to an OpenTelemetry Collector ([#3078](https://github.com/kedacore/keda/issues/3078))
- **General**: New `apache-kafka` scaler based on kafka-go library ([#4692](https://github.com/kedacore/keda/issues/4692))
- **General**: Support for formula-based evaluation of metric values ([#2440](https://github.com/kedacore/keda/issues/2440)|[#4998](https://github.com/kedacore/keda/pull/4998))

### Improvements

- **General**: Automatically set `GOMAXPROCS` to match Linux container CPU quota ([#4999](https://github.com/kedacore/keda/issues/4999))
- **General**: Configurable Kubernetes cluster domain for Cert Manager ([#4861](https://github.com/kedacore/keda/issues/4861))
- **General**: Introduce annotation `autoscaling.keda.sh/paused: true` for ScaledObject to pause autoscaling ([#3304](https://github.com/kedacore/keda/issues/3304))
- **General**: Introduce changelog validation check during PR for formatting and order violations ([#3190](https://github.com/kedacore/keda/issues/3190))
- **General**: Introduce Prometheus metrics of API server to KEDA Metric Server ([#4460](https://github.com/kedacore/keda/issues/4460))
- **General**: Introduce standalone CRD generation to release workflow ([#2726](https://github.com/kedacore/keda/issues/2726))
- **General**: Provide new Kubernetes events about internal state and scaling ([#3764](https://github.com/kedacore/keda/issues/3764))
- **General**: Show ScaledObject/ScaledJob names to output of `kubectl get triggerauthentication/clustertriggerauthentication` ([#796](https://github.com/kedacore/keda/issues/796))
- **General**: Updated AWS SDK and updated all the AWS scalers ([#4905](https://github.com/kedacore/keda/issues/4905))
- **Azure Pod Identity**: Introduce validation to prevent usage of empty identity ID for Azure identity providers ([#4528](https://github.com/kedacore/keda/issues/4528))
- **Kafka Scaler**: Add `unsafeSsl` flag for kafka scaler ([#4977](https://github.com/kedacore/keda/issues/4977))
- **Prometheus Scaler**: Remove trailing whitespaces in `customAuthHeader` and `customAuthValue` ([#4960](https://github.com/kedacore/keda/issues/4960))
- **Pulsar Scaler**: Support for OAuth extensions ([#4700](https://github.com/kedacore/keda/issues/4700))
- **Redis Scalers**: Support for TLS authentication for Redis & Redis stream scalers ([#4917](https://github.com/kedacore/keda/issues/4917))

### Fixes

- **General**: Add validations for `stabilizationWindowSeconds` ([#4976](https://github.com/kedacore/keda/issues/4976))
- **RabbitMQ Scaler**: Allow subpaths along with vhost in connection string ([#2634](https://github.com/kedacore/keda/issues/2634))
- **Selenium Grid Scaler**: Fix scaling based on latest browser version ([#4858](https://github.com/kedacore/keda/issues/4858))
- **Solace Scaler**: Fix a bug where `queueName` is not properly escaped during URL encode ([#4936](https://github.com/kedacore/keda/issues/4936))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- None.

### Breaking Changes

- **General**: Remove deprecated `metricName` from trigger metadata section ([#4899](https://github.com/kedacore/keda/issues/4899))

### Other

- **General**: Fixed a typo in the StatefulSet scaling resolver ([#4902](https://github.com/kedacore/keda/pull/4902))
- **General**: Only show logs with a severity level of ERROR or higher in the stderr in metrics server ([#4049](https://github.com/kedacore/keda/issues/4049))
- **General**: Refactor ScaledJob related methods to be located at scale_handler  ([#4781](https://github.com/kedacore/keda/issues/4781))
- **General**: Replace deprecated `set-output` command with environment file in GitHub Actions workflows ([#4914](https://github.com/kedacore/keda/issues/4914))

## v2.11.2

### Fixes

- **General**: Metrics server exposes Prometheus metrics ([#4776](https://github.com/kedacore/keda/issues/4776))
- **AWS Pod Identity Authentication**: Use `default` service account if the workload doesn't set it ([#4767](https://github.com/kedacore/keda/issues/4767))
- **GitHub Runner Scaler**: Fix explicit repo check 404 to skip not crash ([#4790](https://github.com/kedacore/keda/issues/4790))
- **GitHub Runner Scaler**: Fix rate checking on GHEC when HTTP 200 ([#4786](https://github.com/kedacore/keda/issues/4786))
- **Pulsar Scaler**: Fix `msgBacklogThreshold` field being named wrongly as `msgBacklog` ([#4681](https://github.com/kedacore/keda/issues/4681))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- **Pulsar Scaler**: Fix `msgBacklogThreshold` field being named wrongly as `msgBacklog` ([#4681](https://github.com/kedacore/keda/issues/4681))

## v2.11.1

### New

None.

### Improvements

None.

### Fixes

- **General**: Paused ScaledObject continues working after removing the annotation ([#4733](https://github.com/kedacore/keda/issues/4733))
- **General**: Skip resolving secrets if namespace is restricted ([#4519](https://github.com/kedacore/keda/issues/4519))
- **Prometheus**: Authenticated connections to Prometheus work in non-PodIdenty case ([#4695](https://github.com/kedacore/keda/issues/4695))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s): None.

### Breaking Changes

None.

### Other

None.

## v2.11.0

### New

- **General**: Introduce annotation `autoscaling.keda.sh/paused: true` for ScaledJobs to pause autoscaling ([#3303](https://github.com/kedacore/keda/issues/3303))
- **General**: Introduce new Solr Scaler ([#4234](https://github.com/kedacore/keda/issues/4234))
- **General**: Support ScaledObject taking over existing HPAs with the same name while they are not managed by other ScaledObject ([#4457](https://github.com/kedacore/keda/issues/4457))
- **CPU/Memory scaler**: Add support for scale to zero if there are multiple triggers([#4269](https://github.com/kedacore/keda/issues/4269))
- **Redis Scalers**: Allow scaling using consumer group lag ([#3127](https://github.com/kedacore/keda/issues/3127))
- **Redis Scalers**: Allow scaling using redis stream length ([#4277](https://github.com/kedacore/keda/issues/4277))

### Breaking Changes

- **General**: Metrics Adapter: remove deprecated Prometheus Metrics and non-gRPC code ([#3930](https://github.com/kedacore/keda/issues/3930))

### Improvements

- **General**: Add a Prometheus metric for measuring the processing loop lag ([#4702](https://github.com/kedacore/keda/issues/4702))
- **General**: Add a Prometheus metric with KEDA build info ([#4647](https://github.com/kedacore/keda/issues/4647))
- **General**: Allow to change the port of the Admission Webhook ([#468](https://github.com/kedacore/charts/issues/468))
- **General**: Enable secret scanning in GitHub repo ([#4710](https://github.com/kedacore/keda/issues/4710))
- **General**: Kubernetes v1.25, v1.26 or v1.27 are supported ([#4710](https://github.com/kedacore/keda/issues/4710))
- **AWS DynamoDB**: Add support for `indexName` ([#4680](https://github.com/kedacore/keda/issues/4680))
- **Azure Data Explorer Scaler**: Use azidentity SDK ([#4489](https://github.com/kedacore/keda/issues/4489))
- **External Scaler**: Add tls options in TriggerAuth metadata. ([#3565](https://github.com/kedacore/keda/issues/3565))
- **GCP PubSub Scaler**: Make it more flexible for metrics ([#4243](https://github.com/kedacore/keda/issues/4243))
- **GitHub Runner Scaler**: Added support for GitHub App authentication ([#4651](https://github.com/kedacore/keda/issues/4651))
- **Kafka Scaler**: Add support for OAuth extensions ([#4544](https://github.com/kedacore/keda/issues/4544))
- **NATS JetStream Scaler**: Add support for pulling AccountID from TriggerAuthentication ([#4586](https://github.com/kedacore/keda/issues/4586))
- **PostgreSQL Scaler**: Replace `lib/pq` with `pgx` ([#4704](https://github.com/kedacore/keda/issues/4704))
- **Prometheus Scaler**: Add support for Google Managed Prometheus ([#4674](https://github.com/kedacore/keda/issues/4674))
- **Pulsar Scaler**: Improve error messages for unsuccessful connections ([#4563](https://github.com/kedacore/keda/issues/4563))
- **RabbitMQ Scaler**: Add support for `unsafeSsl` in trigger metadata ([#4448](https://github.com/kedacore/keda/issues/4448))
- **RabbitMQ Scaler**: Add support for `workloadIdentityResource` and utilize AzureAD Workload Identity for HTTP authorization ([#4716](https://github.com/kedacore/keda/issues/4716))
- **Solace Scaler**: Add new `messageReceiveRateTarget` metric to Solace Scaler ([#4665](https://github.com/kedacore/keda/issues/4665))

### Fixes

- **General**: Allow to remove the finalizer even if the ScaledObject isn't valid ([#4396](https://github.com/kedacore/keda/issues/4396))
- **General**: Check ScaledObjects with multiple triggers with non unique name in the Admission Webhook ([#4664](https://github.com/kedacore/keda/issues/4664))
- **General**: Grafana Dashboard: Fix HPA metrics panel by replacing $namepsace to $exported_namespace due to label conflict ([#4539](https://github.com/kedacore/keda/pull/4539))
- **General**: Grafana Dashboard: Fix HPA metrics panel to use range instead of instant ([#4513](https://github.com/kedacore/keda/pull/4513))
- **General**: ScaledJob: Check if MaxReplicaCount is nil before access to it ([#4568](https://github.com/kedacore/keda/issues/4568))
- **AWS SQS Scaler**: Respect `scaleOnInFlight` value ([#4276](https://github.com/kedacore/keda/issues/4276))
- **Azure Monitor**: Exclude Azure Monitor scaler from metricName deprecation ([#4713](https://github.com/kedacore/keda/pull/4713))
- **Azure Pipelines**: Respect all required demands ([#4404](https://github.com/kedacore/keda/issues/4404))
- **Kafka Scaler**: Add back `strings.TrimSpace()` function for saslAuthType ([#4689](https://github.com/kedacore/keda/issues/4689))
- **NATS Jetstream Scaler**: Fix compatibility if node is not advertised ([#4524](https://github.com/kedacore/keda/issues/4524))
- **Prometheus Metrics**: Create e2e tests for all exposed Prometheus metrics ([#4127](https://github.com/kedacore/keda/issues/4127))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- **Azure Data Explorer**: Deprecate `metadata.clientSecret` ([#4514](https://github.com/kedacore/keda/issues/4514))

### Other

- **General**: Add e2e test for external push scaler ([#2698](https://github.com/kedacore/keda/pull/2698))
- **General**: Automatically scale test clusters in/out to reduce environmental footprint & improve cost-efficiency ([#4456](https://github.com/kedacore/keda/pull/4456))
- **General**: Bump Golang to 1.20 ([#4517](https://github.com/kedacore/keda/issues/4517))
- **General**: Bump `kubernetes-sigs/controller-runtime` to v0.15.0 and code alignment ([#4582](https://github.com/kedacore/keda/pull/4582))
- **General**: Drop a transitive dependency on bou.ke/monkey ([#4364](https://github.com/kedacore/keda/issues/4364))
- **General**: Fix odd number of arguments passed as key-value pairs for logging ([#4368](https://github.com/kedacore/keda/issues/4368))
- **General**: Refactor several functions for Status & Conditions handling into pkg util functions ([#2906](https://github.com/kedacore/keda/pull/2906))
- **General**: Stop logging errors for paused ScaledObject (with `autoscaling.keda.sh/paused-replicas` annotation) by skipping reconciliation loop for the object (stop the scale loop and delete the HPA) ([#4253](https://github.com/kedacore/keda/pull/4253))
- **General**: Trying to prevent operator crash when accessing `ScaledObject.Status.ScaleTargetGVKR` ([#4389](https://github.com/kedacore/keda/issues/4389))
- **General**: Use default metrics provider from `sigs.k8s.io/custom-metrics-apiserver` ([#4473](https://github.com/kedacore/keda/pull/4473))

## v2.10.1

### Fixes

- **General**: Drop a transitive dependency on bou.ke/monkey ([#4366](https://github.com/kedacore/keda/issues/4366))
- **General**: Fix odd number of arguments passed as key-value pairs for logging ([#4369](https://github.com/kedacore/keda/issues/4369))
- **General**: Update supported versions in the welcome message ([#4360](https://github.com/kedacore/keda/issues/4360))
- **Admission Webhooks**: Allow to remove the finalizer even if the ScaledObject isn't valid ([#4396](https://github.com/kedacore/keda/issues/4396))
- **AWS SQS Scaler**: Respect `scaleOnInFlight` value ([#4276](https://github.com/kedacore/keda/issues/4276))
- **Azure Pipelines**: Fix for disallowing `$top` on query when using `meta.parentID` method ([#4397](https://github.com/kedacore/keda/issues/4397))
- **Azure Pipelines**: Respect all required demands ([#4404](https://github.com/kedacore/keda/issues/4404))

## v2.10.0

### New

Here is an overview of all **stable** additions:

- **General**: Add support to register custom CAs globally in KEDA operator ([#4168](https://github.com/kedacore/keda/issues/4168))
- **General**: Introduce admission webhooks to automatically validate resource changes to prevent misconfiguration and enforce best practices ([#3755](https://github.com/kedacore/keda/issues/3755))
- **General**: Introduce new ArangoDB Scaler ([#4000](https://github.com/kedacore/keda/issues/4000))
- **Prometheus Metrics**: Introduce scaler activity in Prometheus metrics ([#4114](https://github.com/kedacore/keda/issues/4114))
- **Prometheus Metrics**: Introduce scaler latency in Prometheus metrics ([#4037](https://github.com/kedacore/keda/issues/4037))

#### Experimental

Here is an overview of all new **experimental** features:

- **GitHub Scaler**: Introduced new GitHub Scaler ([#1732](https://github.com/kedacore/keda/issues/1732))

### Improvements

- **General**: Add a warning when KEDA run outside supported k8s versions ([#4130](https://github.com/kedacore/keda/issues/4130))
- **General**: Use (self-signed) certificates for all the communications (internals and externals) ([#3931](https://github.com/kedacore/keda/issues/3931))
- **General**: Use TLS1.2 as minimum TLS version ([#4193](https://github.com/kedacore/keda/issues/4193))
- **Azure Application Insights Scaler**: Add ignoreNullValues to ignore errors when the data returned has null in its values ([#4316](https://github.com/kedacore/keda/issues/4316))
- **Azure Pipelines Scaler**: Improve error logging for `validatePoolID` ([#3996](https://github.com/kedacore/keda/issues/3996))
- **Azure Pipelines Scaler**: New configuration parameter `requireAllDemands` to scale only if jobs request all demands provided by the scaling definition ([#4138](https://github.com/kedacore/keda/issues/4138))
- **Hashicorp Vault**: Add support to secrets backend version 1 ([#2645](https://github.com/kedacore/keda/issues/2645))
- **Kafka Scaler**: Add support to use `tls` and `sasl` in ScaledObject ([#4322](https://github.com/kedacore/keda/issues/4322))
- **Kafka Scaler**: Improve error logging for `GetBlock` method ([#4232](https://github.com/kedacore/keda/issues/4232))
- **Prometheus Scaler**: Add custom headers and custom auth support ([#4208](https://github.com/kedacore/keda/issues/4208))
- **Prometheus Scaler**: Extend Prometheus Scaler to support Azure managed service for Prometheus ([#4153](https://github.com/kedacore/keda/issues/4153))
- **RabbitMQ Scaler**:  Add TLS support ([#967](https://github.com/kedacore/keda/issues/967))
- **Redis Scalers**: Add support to Redis 7 ([#4052](https://github.com/kedacore/keda/issues/4052))
- **Selenium Grid Scaler**: Add `platformName` to selenium-grid scaler metadata structure ([#4038](https://github.com/kedacore/keda/issues/4038))

### Fixes

- **General**: Fix regression in fallback mechanism ([#4249](https://github.com/kedacore/keda/issues/4249))
- **General**: Prevent a panic that might occur while refreshing a scaler cache ([#4092](https://github.com/kedacore/keda/issues/4092))
- **AWS Cloudwatch Scaler**: Make `metricName` and `namespace` optional when using `expression` ([#4334](https://github.com/kedacore/keda/issues/4334))
- **Azure Pipelines Scaler**: Add new parameter to limit the jobs returned ([#4324](https://github.com/kedacore/keda/issues/4324))
- **Azure Queue Scaler**: Fix azure queue length ([#4002](https://github.com/kedacore/keda/issues/4002))
- **Azure Service Bus Scaler**: Improve way clients are created to reduce amount of ARM requests ([#4262](https://github.com/kedacore/keda/issues/4262))
- **Azure Service Bus Scaler**: Use correct auth flows with pod identity ([#4026](https://github.com/kedacore/keda/issues/4026)|[#4123](https://github.com/kedacore/keda/issues/4123))
- **Cassandra Scaler**: Checking whether the port information is entered in the ClusterIPAddres is done correctly. ([#4110](https://github.com/kedacore/keda/issues/4110))
- **CPU Memory Scaler**: Store forgotten logger ([#4022](https://github.com/kedacore/keda/issues/4022))
- **Datadog Scaler**: Return correct error when getting a 429 error ([#4187](https://github.com/kedacore/keda/issues/4187))
- **Kafka Scaler**: Return error if the processing of the partition lag fails ([#4098](https://github.com/kedacore/keda/issues/4098))
- **Kafka Scaler**: Support 0 in activationLagThreshold configuration ([#4137](https://github.com/kedacore/keda/issues/4137))
- **Kafka Scaler**: Trim whitespace from `partitionLimitation` field ([#4333](https://github.com/kedacore/keda/pull/4333))
- **NATS Jetstream Scaler**: Fix compatibility when cluster not on kubernetes ([#4101](https://github.com/kedacore/keda/issues/4101))
- **Prometheus Metrics**: Expose Prometheus Metrics also when getting ScaledObject state ([#4075](https://github.com/kedacore/keda/issues/4075))
- **Redis Scalers**: Fix panic produced by incorrect logger initialization ([#4197](https://github.com/kedacore/keda/issues/4197))
- **Selenium Grid Scaler**: ScaledObject with a trigger whose metadata browserVersion is latest is always being triggered regardless of the browserVersion requested by the user ([#4347](https://github.com/kedacore/keda/issues/4347))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- **General**: Deprecate explicitly setting `metricName` field from `ScaledObject.triggers[*].metadata` ([#4220](https://github.com/kedacore/keda/issues/4220))
- **Prometheus Scaler**: `cortexOrgId` metadata deprecated in favor of custom headers ([#4208](https://github.com/kedacore/keda/issues/4208))

### Other

- **General**: Bump Golang to 1.19 ([#4094](https://github.com/kedacore/keda/issues/4094))
- **General**: Check that ScaledObject name is specified as part of a query for getting metrics ([#4088](https://github.com/kedacore/keda/pull/4088))
- **General**: Compare error with `errors.Is` ([#4004](https://github.com/kedacore/keda/pull/4004))
- **General**: Consolidate `GetMetrics` and `IsActive` to `GetMetricsAndActivity` for Azure Event Hub, Cron and External scalers ([#4015](https://github.com/kedacore/keda/issues/4015))
- **General**: Improve test coverage in `pkg/util` ([#3871](https://github.com/kedacore/keda/issues/3871))
- **General**: Metrics Server: print a message on successful connection to gRPC server ([#4190](https://github.com/kedacore/keda/issues/4190))
- **General**: Pass deep copy object to scalers cache from the ScaledObject controller ([#4207](https://github.com/kedacore/keda/issues/4207))
- **General**: Review CodeQL rules and enable it on PRs ([#4032](https://github.com/kedacore/keda/pull/4032))
- **RabbitMQ Scaler**: Move from `streadway/amqp` to `rabbitmq/amqp091-go` ([#4004](https://github.com/kedacore/keda/issues/4004))

## v2.9.3

### Fixes

- **Azure Service Bus Scaler**: Use correct auth flows with pod identity ([#4026](https://github.com/kedacore/keda/issues/4026)|[#4123](https://github.com/kedacore/keda/issues/4123))

## v2.9.2

### Fixes

- **General**: Prevent a panic that might occur while refreshing a scaler cache ([#4092](https://github.com/kedacore/keda/issues/4092))
- **Azure Service Bus Scaler**: Use correct auth flows with pod identity ([#4026](https://github.com/kedacore/keda/issues/4026))
- **Prometheus Metrics**: Fix exposed metric from `keda_scaled_errors` to `keda_scaled_object_errors` ([#4037](https://github.com/kedacore/keda/issues/4037))

## v2.9.1

### Fixes

- **General**: Properly retrieve and close scalers cache ([#4011](https://github.com/kedacore/keda/issues/4011))
- **Azure Key Vault**: Raise an error if authentication mechanism not provided ([#4010](https://github.com/kedacore/keda/issues/4010))
- **Redis Scalers**: Support `unsafeSsl` and enable ssl verification as default ([#4005](https://github.com/kedacore/keda/issues/4005))

## v2.9.0

### Breaking Changes

- **General**: Change API version of HPA from `autoscaling/v2beta2` to `autoscaling/v2` ([#2462](https://github.com/kedacore/keda/issues/2462))
- **General**: As per our [support policy](https://github.com/kedacore/governance/blob/main/SUPPORT.md), Kubernetes v1.23 or above is required and support for Kubernetes v1.22 or below was removed ([docs](https://keda.sh/docs/2.9/operate/cluster/#kubernetes-compatibility))

### New

Here is an overview of all **stable** additions:

- **General**: Introduce new CouchDB Scaler ([#3746](https://github.com/kedacore/keda/issues/3746))
- **General**: Introduce new Etcd Scaler ([#3880](https://github.com/kedacore/keda/issues/3880))
- **General**: Introduce new Loki Scaler ([#3699](https://github.com/kedacore/keda/issues/3699))
- **General**: Introduce rate-limitting parameters to KEDA manager to allow override of client defaults ([#3730](https://github.com/kedacore/keda/issues/3730))
- **General**: Introduction deprecation & breaking change policy ([#68](https://github.com/kedacore/governance/issues/68))
- **General**: Produce reproducible builds ([#3509](https://github.com/kedacore/keda/issues/3509))
- **General**: Provide off-the-shelf Grafana dashboard for application autoscaling ([#3911](https://github.com/kedacore/keda/issues/3911))
- **AWS Scalers**: Introduce new AWS endpoint URL settings. ([#3337](https://github.com/kedacore/keda/issues/3337))
- **Azure Service Bus Scaler**: Support for Shared Access Signature (SAS) tokens for authentication. ([#2920](https://github.com/kedacore/keda/issues/2920))
- **Azure Service Bus Scaler**: Support regex usage in queueName / subscriptionName parameters. ([#1624](https://github.com/kedacore/keda/issues/1624))
- **ElasticSearch Scaler**: Support for ElasticSearch Service on Elastic Cloud ([#3785](https://github.com/kedacore/keda/issues/3785))
- **Prometheus Metrics**: Expose renamed version of existing Prometheus Metrics in KEDA Operator. ([#3919](https://github.com/kedacore/keda/issues/3919))
- **Prometheus Metrics**: Introduce new `ScalerName` label in Prometheus metrics. ([#3588](https://github.com/kedacore/keda/issues/3588))
- **Prometheus Metrics**: Provide Prometheus metric with indication of total number of custom resources per namespace for each custom resource type (CRD). ([#2637](https://github.com/kedacore/keda/issues/2637)|[#2638](https://github.com/kedacore/keda/issues/2638)|[#2639](https://github.com/kedacore/keda/issues/2639))
- **Prometheus Metrics**: Provide Prometheus metric with indication of total number of triggers per trigger type in `ScaledJob`/`ScaledObject`. ([#3663](https://github.com/kedacore/keda/issues/3663))
- **Selenium Grid Scaler**: Allow setting url trigger parameter from TriggerAuthentication/ClusterTriggerAuthentication ([#3752](https://github.com/kedacore/keda/pull/3752))

#### Experimental

Here is an overview of all new **experimental** features:

- **General**: Adding an option to cache metric values for a scaler during the polling interval ([#2282](https://github.com/kedacore/keda/issues/2282))

### Improvements

- **General**: Add explicit `seccompProfile` type to `securityContext` config ([#3561](https://github.com/kedacore/keda/issues/3561))
- **General**: Add `Min` column to ScaledJob visualization ([#3689](https://github.com/kedacore/keda/issues/3689))
- **General**: Disable response compression for k8s restAPI in client-go ([#3863](https://github.com/kedacore/keda/issues/3863))
- **General**: Improve the function used to normalize metric names ([#3789](https://github.com/kedacore/keda/issues/3789))
- **General**: Support disable keep http connection alive ([#3874](https://github.com/kedacore/keda/issues/3874))
- **General**: Support for using pod identities for authentication in Azure Key Vault ([#3813](https://github.com/kedacore/keda/issues/3813))
- **General**: Support "Restrict Secret Access" to mitigate the security risk ([#3668](https://github.com/kedacore/keda/issues/3668))
- **Apache Kafka Scaler**: Support for excluding persistent lag ([#3904](https://github.com/kedacore/keda/issues/3904))
- **Apache Kafka Scaler**: Support for limiting Kafka partitions KEDA will monitor ([#3830](https://github.com/kedacore/keda/issues/3830))
- **Apache Kafka Scaler**: Support for SASL/OAuth bearer authentication ([#3681](https://github.com/kedacore/keda/issues/3681))
- **Azure AD Pod Identity Authentication**: Improve logs around integration with aad-pod-identity for simplified troubleshooting ([#3610](https://github.com/kedacore/keda/issues/3610))
- **Azure Event Hubs Scaler**: Support Azure Active Directory Pod & Workload Identity for Storage Blobs ([#3569](https://github.com/kedacore/keda/issues/3569))
- **Azure Event Hubs Scaler**: Support for `dapr` checkpoint strategy ([#3022](https://github.com/kedacore/keda/issues/3022))
- **Azure Event Hubs Scaler**: Support for using connection strings for Event Hub namespace instead of the Event Hub itself. ([#3922](https://github.com/kedacore/keda/issues/3922))
- **Azure Pipelines Scaler**: Improved performance for scaling big amount of job requests ([#3702](https://github.com/kedacore/keda/issues/3702))
- **Cron Scaler**: Improve instance count determination. ([#3838](https://github.com/kedacore/keda/issues/3838))
- **GCP Storage Scaler**: Support for blob delimiters ([#3756](https://github.com/kedacore/keda/issues/3756))
- **GCP Storage Scaler**: Support for blob prefix ([#3756](https://github.com/kedacore/keda/issues/3756))
- **Metrics API Scaler**: Support for `unsafeSsl` parameter to skip certificate validation when connecting over HTTPS ([#3728](https://github.com/kedacore/keda/discussions/3728))
- **NATS Jetstream Scaler**: Improved querying to respect stream consumer leader in clustered scenarios ([#3860](https://github.com/kedacore/keda/issues/3860))
- **NATS Scalers**: Support HTTPS protocol in NATS Scalers ([#3805](https://github.com/kedacore/keda/issues/3805))
- **Prometheus Scaler**: Introduce skipping of certificate check for unsigned certs ([#2310](https://github.com/kedacore/keda/issues/2310))
- **Pulsar Scaler**: Add support for basic authentication ([#3844](https://github.com/kedacore/keda/issues/3844))
- **Pulsar Scaler**: Add support for bearer token authentication ([#3844](https://github.com/kedacore/keda/issues/3844))
- **Pulsar Scaler**: Add support for partitioned topics ([#3833](https://github.com/kedacore/keda/issues/3833))

### Fixes

- **General**: Ensure `Close` is only called once during `PushScaler`'s deletion ([#3881](https://github.com/kedacore/keda/issues/3881))
- **General**: Respect optional parameter inside `envs` for ScaledJobs ([#3568](https://github.com/kedacore/keda/issues/3568))
- **Azure Blob Scaler**: Store forgotten logger ([#3811](https://github.com/kedacore/keda/issues/3811))
- **Datadog Scaler**: The last data point of some specific query is always null ([#3906](https://github.com/kedacore/keda/issues/3906))
- **GCP Stackdriver Scalar**: Update Stackdriver client to handle detecting double and int64 value types ([#3777](https://github.com/kedacore/keda/issues/3777))
- **MongoDB Scaler**: Username/password can contain `:/?#[]@` ([#3992](https://github.com/kedacore/keda/issues/3992))
- **NATS Jetstream**: Correctly count messages that should be redelivered (waiting for ack) towards KEDA value ([#3787](https://github.com/kedacore/keda/issues/3787))
- **New Relic Scaler**: Store forgotten logger ([#3945](https://github.com/kedacore/keda/issues/3945))
- **Prometheus Scaler**: Treat Inf the same as Null result ([#3644](https://github.com/kedacore/keda/issues/3644))
- **Security**: Provide patch for CVE-2022-3172 vulnerability ([#3690](https://github.com/kedacore/keda/issues/3690))

### Deprecations

You can find all deprecations in [this overview](https://github.com/kedacore/keda/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3Abreaking-change) and [join the discussion here](https://github.com/kedacore/keda/discussions/categories/deprecations).

New deprecation(s):

- Prometheus metrics on KEDA Metric Server are deprecated in favor of Prometheus metrics on KEDA Operator ([#3972](https://github.com/kedacore/keda/issues/3972) | [Discussion](https://github.com/kedacore/keda/discussions/3973))

Previously announced deprecation(s):

- Default checkpointing strategy for Azure Event Hubs scaler `azureFunctions` is deprecated in favor of `blobMetadata` ([#XXX](https://github.com/kedacore/keda/issues/3596) | [Discussion](https://github.com/kedacore/keda/discussions/3552))
- `metadata.type` is deprecated in favor of the global `metricType` for CPU, Memory, Datadog scalers ([#2844](https://github.com/kedacore/keda/issues/2844) | [Discussion](https://github.com/kedacore/keda/discussions/3002))
- `rolloutStrategy` is deprecated in favor of `rollout.strategy` in ScaledJobs ([#3596](https://github.com/kedacore/keda/issues/3596) | [Discussion](https://github.com/kedacore/keda/discussions/3552))

### Other

- **General**: Bump `github.com/Azure/azure-event-hubs-go/v3` ([#2986](https://github.com/kedacore/keda/issues/2986))
- **General**: Bump Golang to 1.18.6 ([#3205](https://github.com/kedacore/keda/issues/3205))
- **General**: Metrics Server: use gRPC connection to get metrics from Operator ([#3920](https://github.com/kedacore/keda/issues/3920))
- **General**: Metrics Server: use OpenAPI definitions served by custom-metrics-apiserver ([#3929](https://github.com/kedacore/keda/issues/3929))
- **General**: Migrate from `azure-service-bus-go` to `azservicebus` ([#3394](https://github.com/kedacore/keda/issues/3394))
- **Apache Kafka Scaler**: Increase logging V-level  ([#3948](https://github.com/kedacore/keda/issues/3948))
- **Azure EventHub**: Add e2e tests ([#2792](https://github.com/kedacore/keda/issues/2792))

## v2.8.1

### New

None.

### Improvements

- **Datadog Scaler**: Support multi-query metrics, and aggregation ([#3423](https://github.com/kedacore/keda/issues/3423))

### Fixes

- **General**: Metrics endpoint returns correct HPA values ([#3554](https://github.com/kedacore/keda/issues/3554))
- **Datadog Scaler**: Fix: panic in datadog scaler ([#3448](https://github.com/kedacore/keda/issues/3448))
- **RabbitMQ Scaler**: Parse vhost correctly if it's provided in the host url ([#3602](https://github.com/kedacore/keda/issues/3602))

### Deprecations

None.

### Breaking Changes

None.

### Other

- **General**: Execute trivy scan (on PRs) only if there are changes in deps ([#3540](https://github.com/kedacore/keda/issues/3540))
- **General**: Use re-usable workflows for GitHub Actions ([#2569](https://github.com/kedacore/keda/issues/2569))

## v2.8.0

### New

- **General**: Introduce `activationThreshold`/`minMetricValue` for all scalers ([#2800](https://github.com/kedacore/keda/issues/2800))
- **General**: Introduce new AWS DynamoDB Streams Scaler ([#3124](https://github.com/kedacore/keda/issues/3124))
- **General**: Introduce new NATS JetStream scaler ([#2391](https://github.com/kedacore/keda/issues/2391))
- **General**: Make propagation policy for ScaledJob rollout configurable ([#2910](https://github.com/kedacore/keda/issues/2910))
- **General**: Support for Azure AD Workload Identity as a pod identity provider. ([#2487](https://github.com/kedacore/keda/issues/2487)|[#2656](https://github.com/kedacore/keda/issues/2656))
- **General**: Support for `minReplicaCount` in ScaledJob ([#3426](https://github.com/kedacore/keda/issues/3426))
- **General**: Support for permission segregation when using Azure AD Pod / Workload Identity. ([#2656](https://github.com/kedacore/keda/issues/2656))
- **General**: Support to customize HPA name ([#3057](https://github.com/kedacore/keda/issues/3057))
- **AWS SQS Queue Scaler**: Support for scaling to include in-flight messages. ([#3133](https://github.com/kedacore/keda/issues/3133))
- **Azure Pipelines Scaler**: Support for Azure Pipelines to support demands (capabilities) ([#2328](https://github.com/kedacore/keda/issues/2328))
- **CPU Scaler**: Support for targeting specific container in a pod ([#1378](https://github.com/kedacore/keda/issues/1378))
- **GCP Stackdriver Scaler**: Added aggregation parameters ([#3008](https://github.com/kedacore/keda/issues/3008))
- **Kafka Scaler**: Support of passphrase encrypted PKCS #\8 private key ([#3449](https://github.com/kedacore/keda/issues/3449))
- **Memory Scaler**: Support for targeting specific container in a pod ([#1378](https://github.com/kedacore/keda/issues/1378))
- **Prometheus Scaler**: Add `ignoreNullValues` to return error when prometheus return null in values ([#3065](https://github.com/kedacore/keda/issues/3065))

### Improvements

- **General**: Add settings for configuring leader election ([#2836](https://github.com/kedacore/keda/issues/2836))
- **General**: `external` extension reduces connection establishment with long links ([#3193](https://github.com/kedacore/keda/issues/3193))
- **General**: Reference ScaledObject's/ScaledJob's name in the scalers log ([#3419](https://github.com/kedacore/keda/issues/3419))
- **General**: Use `mili` scale for the returned metrics ([#3135](https://github.com/kedacore/keda/issues/3135))
- **General**: Use more readable timestamps in KEDA Operator logs ([#3066](https://github.com/kedacore/keda/issues/3066))
- **Kafka Scaler**: Handle Sarama errors properly ([#3056](https://github.com/kedacore/keda/issues/3056))

### Fixes

- **General**: Provide patch for CVE-2022-27191 vulnerability ([#3378](https://github.com/kedacore/keda/issues/3378))
- **General**: Refactor adapter startup to ensure proper log initilization. ([#2316](https://github.com/kedacore/keda/issues/2316))
- **General**: Scaleobject ready condition 'False/Unknown' to 'True' requeue ([#3096](https://github.com/kedacore/keda/issues/3096))
- **General**: Use `go install` in the Makefile for downloading dependencies ([#2916](https://github.com/kedacore/keda/issues/2916))
- **General**: Use metricName from GetMetricsSpec in ScaledJobs instead of `queueLength` ([#3032](https://github.com/kedacore/keda/issues/3032))
- **ActiveMQ Scaler**: KEDA doesn't respect restAPITemplate ([#3188](https://github.com/kedacore/keda/issues/3188))
- **Azure Eventhub Scaler**: KEDA operator crashes on nil memory panic if the eventhub connectionstring for Azure Eventhub Scaler contains an invalid character ([#3082](https://github.com/kedacore/keda/issues/3082))
- **Azure Pipelines Scaler**: Fix issue with Azure Pipelines wrong PAT Auth. ([#3159](https://github.com/kedacore/keda/issues/3159))
- **Datadog Scaler**: Ensure that returns the same element that has been checked ([#3448](https://github.com/kedacore/keda/issues/3448))
- **Kafka Scaler**: Check `lagThreshold` is a positive number ([#3366](https://github.com/kedacore/keda/issues/3366))
- **Selenium Grid Scaler**: Fix bug where edge active sessions not being properly counted ([#2709](https://github.com/kedacore/keda/issues/2709))
- **Selenium Grid Scaler**: Fix bug where Max Sessions was not working correctly ([#3061](https://github.com/kedacore/keda/issues/3061))

### Deprecations

- **ScaledJob**: `rolloutStrategy` is deprecated in favor of `rollout.strategy` ([#2910](https://github.com/kedacore/keda/issues/2910))

### Breaking Changes

None.

### Other

- **General**: Bump Golang to 1.17.13 and deps ([#3447](https://github.com/kedacore/keda/issues/3447))
- **General**: Fix devcontainer on ARM64 Arch. ([#3084](https://github.com/kedacore/keda/issues/3084))
- **General**: Improve e2e on PR process through comments. ([#3004](https://github.com/kedacore/keda/issues/3004))
- **General**: Improve error message in resolving ServiceAccount for AWS EKS PodIdentity ([#3142](https://github.com/kedacore/keda/issues/3142))
- **General**: Migrate e2e test to Go. ([#2737](https://github.com/kedacore/keda/issues/2737))
- **General**: Split e2e test by functionality. ([#3270](https://github.com/kedacore/keda/issues/3270))
- **General**: Unify the used tooling on different workflows and arch. ([#3092](https://github.com/kedacore/keda/issues/3092))
- **General**: Use Github's Checks API for e2e tests on PR. ([#2567](https://github.com/kedacore/keda/issues/2567))

## v2.7.1

### Improvements

- **General**: Don't hardcode UIDs in securityContext ([#3012](https://github.com/kedacore/keda/issues/3012))

### Other

- **General**: Bump Golang to 1.17.9 ([#3016](https://github.com/kedacore/keda/issues/3016))
- **General**: Fix autoscaling behaviour while paused. ([#3009](https://github.com/kedacore/keda/issues/3009))
- **General**: Fix CVE-2022-21221 in `github.com/valyala/fasthttp` ([#2775](https://github.com/kedacore/keda/issues/2775))

## v2.7.0

### New

- **General**: Introduce annotation `"autoscaling.keda.sh/paused-replicas"` for ScaledObjects to pause scaling at a fixed replica count. ([#944](https://github.com/kedacore/keda/issues/944))
- **General**: Introduce ARM-based container image for KEDA ([#2263](https://github.com/kedacore/keda/issues/2263)|[#2262](https://github.com/kedacore/keda/issues/2262))
- **General**: Introduce new AWS DynamoDB Scaler ([#2482](https://github.com/kedacore/keda/issues/2482))
- **General**: Introduce new Azure Data Explorer Scaler ([#1488](https://github.com/kedacore/keda/issues/1488)|[#2734](https://github.com/kedacore/keda/issues/2734))
- **General**: Introduce new GCP Stackdriver Scaler ([#2661](https://github.com/kedacore/keda/issues/2661))
- **General**: Introduce new GCP Storage Scaler ([#2628](https://github.com/kedacore/keda/issues/2628))
- **General**: Provide support for authentication via Azure Key Vault ([#900](https://github.com/kedacore/keda/issues/900)|[#2733](https://github.com/kedacore/keda/issues/2733))
- **General**: Support for `ValueMetricType` in `ScaledObject` for all scalers except CPU/Memory ([#2030](https://github.com/kedacore/keda/issues/2030))

### Improvements

- **General**: Bump dependencies versions ([#2978](https://github.com/kedacore/keda/issues/2978))
- **General**: Properly handle `restoreToOriginalReplicaCount` if `ScaleTarget` is missing ([#2872](https://github.com/kedacore/keda/issues/2872))
- **General**: Support for running KEDA secure-by-default as non-root ([#2933](https://github.com/kedacore/keda/issues/2933))
- **General**: Synchronize HPA annotations from ScaledObject ([#2659](https://github.com/kedacore/keda/pull/2659))
- **General**: Updated HTTPClient to be proxy-aware, if available, from environment variables. ([#2577](https://github.com/kedacore/keda/issues/2577))
- **General**: Using manager client in KEDA Metrics Server to avoid flush request to Kubernetes Apiserver([#2914](https://github.com/kedacore/keda/issues/2914))
- **ActiveMQ Scaler**: Add CorsHeader information to ActiveMQ Scaler ([#2884](https://github.com/kedacore/keda/issues/2884))
- **AWS CloudWatch**: Add support to use expressions([#2998](https://github.com/kedacore/keda/issues/2998))
- **Azure Application Insights Scaler**: Provide support for non-public clouds ([#2735](https://github.com/kedacore/keda/issues/2735))
- **Azure Blob Storage Scaler**: Add optional parameters for counting blobs recursively ([#1789](https://github.com/kedacore/keda/issues/1789))
- **Azure Event Hub Scaler**: Improve logging when blob container not found ([#2363](https://github.com/kedacore/keda/issues/2363))
- **Azure Event Hub Scaler**: Provide support for non-public clouds ([#1915](https://github.com/kedacore/keda/issues/1915))
- **Azure Log Analytics Scaler**: Provide support for non-public clouds ([#1916](https://github.com/kedacore/keda/issues/1916))
- **Azure Monitor Scaler**: Provide support for non-public clouds ([#1917](https://github.com/kedacore/keda/issues/1917))
- **Azure Queue**: Don't call Azure queue GetProperties API unnecessarily ([#2613](https://github.com/kedacore/keda/pull/2613))
- **Datadog Scaler**: Rely on Datadog API to validate the query ([#2761](https://github.com/kedacore/keda/issues/2761))
- **Datadog Scaler**: Several improvements, including a new optional parameter `metricUnavailableValue` to fill data when no Datadog metric was returned ([#2657](https://github.com/kedacore/keda/issues/2657))
- **Datadog Scaler**: Validate query to contain `{` to prevent panic on invalid query ([#2625](https://github.com/kedacore/keda/issues/2625))
- **Graphite Scaler**: Use the latest non-null datapoint returned by query ([#2944](https://github.com/kedacore/keda/issues/2944))
- **Kafka Scaler**: Make "disable" a valid value for tls auth parameter ([#2608](https://github.com/kedacore/keda/issues/2608))
- **Kafka Scaler**: New `scaleToZeroOnInvalidOffset` to control behavior when partitions have an invalid offset ([#2033](https://github.com/kedacore/keda/issues/2033)|[#2612](https://github.com/kedacore/keda/issues/2612))
- **Metric API Scaler**: Improve error handling on not-ok response ([#2317](https://github.com/kedacore/keda/issues/2317))
- **New Relic Scaler**: Support to get account value from authentication resources. ([#2883](https://github.com/kedacore/keda/issues/2883))
- **Prometheus Scaler**: Check and properly inform user that `threshold` is not set ([#2793](https://github.com/kedacore/keda/issues/2793))
- **Prometheus Scaler**: Support for `X-Scope-OrgID` header ([#2667](https://github.com/kedacore/keda/issues/2667))
- **RabbitMQ Scaler**: Include `vhost` for RabbitMQ when retrieving queue info with `useRegex` ([#2498](https://github.com/kedacore/keda/issues/2498))
- **Selenium Grid Scaler**: Consider `maxSession` grid info when scaling. ([#2618](https://github.com/kedacore/keda/issues/2618))

### Deprecations

- **CPU, Memory, Datadog Scalers**: `metadata.type` is deprecated in favor of the global `metricType` ([#2030](https://github.com/kedacore/keda/issues/2030))

### Breaking Changes

None.

### Other

- **General**: Clean go.mod to fix golangci-lint ([#2783](https://github.com/kedacore/keda/issues/2783))
- **General**: Consistent file naming in `pkg/scalers/` ([#2806](https://github.com/kedacore/keda/issues/2806))
- **General**: Fix mismatched errors for updating HPA ([#2719](https://github.com/kedacore/keda/issues/2719))
- **General**: Improve e2e tests reliability ([#2580](https://github.com/kedacore/keda/issues/2580))
- **General**: Improve e2e tests to always cleanup resources in cluster ([#2584](https://github.com/kedacore/keda/issues/2584))
- **General**: Internally represent value and threshold as int64 ([#2790](https://github.com/kedacore/keda/issues/2790))
- **General**: Refactor active directory endpoint parsing for Azure scalers. ([#2853](https://github.com/kedacore/keda/pull/2853))
- **AWS CloudWatch**: Adding e2e test ([#1525](https://github.com/kedacore/keda/issues/1525))
- **AWS DynamoDB**: Setup AWS DynamoDB test account ([#2803](https://github.com/kedacore/keda/issues/2803))
- **AWS Kinesis Stream**: Adding e2e test ([#1526](https://github.com/kedacore/keda/issues/1526))
- **AWS SQS Queue**: Adding e2e test ([#1527](https://github.com/kedacore/keda/issues/1527))
- **Azure Data Explorer**: Adding e2e test ([#2841](https://github.com/kedacore/keda/issues/2841))
- **Azure Data Explorer**: Replace deprecated function `iter.Next()` in favour of `iter.NextRowOrError()` ([#2989](https://github.com/kedacore/keda/issues/2989))
- **Azure Service Bus**: Adding e2e test ([#2731](https://github.com/kedacore/keda/issues/2731)|[#2732](https://github.com/kedacore/keda/issues/2732))
- **External Scaler**: Adding e2e test. ([#2697](https://github.com/kedacore/keda/issues/2697))
- **External Scaler**: Fix issue with internal KEDA core prefix being passed to external scaler. ([#2640](https://github.com/kedacore/keda/issues/2640))
- **GCP Pubsub Scaler**: Adding e2e test ([#1528](https://github.com/kedacore/keda/issues/1528))
- **Hashicorp Vault Secret Provider**: Adding e2e test ([#2842](https://github.com/kedacore/keda/issues/2842))
- **Memory Scaler**: Adding e2e test ([#2220](https://github.com/kedacore/keda/issues/2220))
- **Selenium Grid Scaler**: Adding e2e test ([#2791](https://github.com/kedacore/keda/issues/2791))

## v2.6.1

### Improvements

- **General**: Fix generation of metric names if any of ScaledObject's triggers is unavailable ([#2592](https://github.com/kedacore/keda/issues/2592))
- **General**: Fix logging in KEDA operator and properly set `ScaledObject.Status` in case there is a problem in a ScaledObject's trigger ([#2603](https://github.com/kedacore/keda/issues/2603))

### Other

- **General**: Fix failing tests based on the scale to zero bug ([#2603](https://github.com/kedacore/keda/issues/2603))

## v2.6.0

### New

- Add ActiveMQ Scaler ([#2305](https://github.com/kedacore/keda/pull/2305))
- Add Azure Application Insights Scaler ([2506](https://github.com/kedacore/keda/pull/2506))
- Add New Datadog Scaler ([#2354](https://github.com/kedacore/keda/pull/2354))
- Add New Relic Scaler ([#2387](https://github.com/kedacore/keda/pull/2387))
- Add PredictKube Scaler ([#2418](https://github.com/kedacore/keda/pull/2418))

### Improvements

- **General**: Delete the cache entry when a ScaledObject is deleted ([#2564](https://github.com/kedacore/keda/pull/2564))
- **General**: Fail fast on `buildScalers` when not able to resolve a secret that a deployment is relying on ([#2394](https://github.com/kedacore/keda/pull/2394))
- **General**: `keda-operator` Cluster Role: add `list` and `watch` access to service accounts ([#2406](https://github.com/kedacore/keda/pull/2406))|([#2410](https://github.com/kedacore/keda/pull/2410))
- **General**: Sign KEDA images published on GitHub Container Registry ([#2501](https://github.com/kedacore/keda/pull/2501))|([#2502](https://github.com/kedacore/keda/pull/2502))|([#2504](https://github.com/kedacore/keda/pull/2504))
- **AWS Scalers**: Support temporary AWS credentials using session tokens ([#2573](https://github.com/kedacore/keda/pull/2573))
- **AWS SQS Scaler**: Allow using simple queue name instead of URL ([#2483](https://github.com/kedacore/keda/pull/2483))
- **Azure EventHub Scaler**: Don't expose connection string in metricName ([#2404](https://github.com/kedacore/keda/pull/2404))
- **Azure Pipelines Scaler**: Support `poolName` or `poolID` validation ([#2370](https://github.com/kedacore/keda/pull/2370))
- **CPU Scaler**: Adding e2e test for the cpu scaler ([#2441](https://github.com/kedacore/keda/pull/2441))
- **External Scaler**: Fix wrong calculation of retry backoff duration ([#2416](https://github.com/kedacore/keda/pull/2416))
- **Graphite Scaler**: Use the latest datapoint returned, not the earliest ([#2365](https://github.com/kedacore/keda/pull/2365))
- **Kafka Scaler**: Allow flag `topic` to be optional, where lag of all topics within the consumer group will be used for scaling ([#2409](https://github.com/kedacore/keda/pull/2409))
- **Kafka Scaler**: Concurrently query brokers for consumer and producer offsets ([#2405](https://github.com/kedacore/keda/pull/2405))
- **Kubernetes Workload Scaler**: Ignore terminated pods ([#2384](https://github.com/kedacore/keda/pull/2384))
- **PostgreSQL Scaler**: Assign PostgreSQL `userName` to correct attribute ([#2432](https://github.com/kedacore/keda/pull/2432))|([#2433](https://github.com/kedacore/keda/pull/2433))
- **Prometheus Scaler**: Support namespaced Prometheus queries ([#2575](https://github.com/kedacore/keda/issues/2575))

### Breaking Changes

- No longer push to Docker Hub as of v2.5 as per our [announcement in March 2021](https://github.com/kedacore/keda/discussions/1700)
  - Learn more about the background on [kedacore/governance#16](https://github.com/kedacore/governance/issues/16)

## v2.5.0

### New

- Add Cassandra Scaler ([#2211](https://github.com/kedacore/keda/pull/2211))
- Add Elasticsearch Scaler ([#2311](https://github.com/kedacore/keda/pull/2311))
- Add Graphite Scaler ([#1628](https://github.com/kedacore/keda/pull/2092))
- ScaledJob: introduce `MultipleScalersCalculation` ([#2016](https://github.com/kedacore/keda/pull/2016))
- ScaledJob: introduce `RolloutStrategy` ([#2164](https://github.com/kedacore/keda/pull/2164))
- Add ScalersCache to reuse scalers unless they need changing ([#2187](https://github.com/kedacore/keda/pull/2187))
- Cache metric names provided by KEDA Metrics Server ([#2279](https://github.com/kedacore/keda/pull/2279))

### Improvements

- Artemis Scaler: parse out broker config parameters in case `restAPITemplate` is given ([#2104](https://github.com/kedacore/keda/pull/2104))
- AWS Cloudwatch Scaler: improve metric exporting logic ([#2243](https://github.com/kedacore/keda/pull/2243))
- AWS Cloudwatch Scaler: return minimum value for the metric when cloudwatch returns empty list ([#2345](https://github.com/kedacore/keda/pull/2345))
- Azure Log Analytics Scaler: add support to provide the metric name([#2106](https://github.com/kedacore/keda/pull/2106))
- Azure Pipelines Scaler: improve logs ([#2297](https://github.com/kedacore/keda/pull/2297))
- Cron Scaler: improve validation in case start & end input is same ([#2032](https://github.com/kedacore/keda/pull/2032))
- Cron Scaler: improve the cron validation ([#2038](https://github.com/kedacore/keda/pull/2038))
- GCP PubSub Scaler: introduce `SubscriptionSize` and `OldestUnackedMessageAge` modes ([#2266](https://github.com/kedacore/keda/pull/2266))
- GCP PubSub Scaler: add GCP identity authentication when using ([#2225](https://github.com/kedacore/keda/pull/2225))
- GCP PubSub Scaler: add possibility to reference a GCP PubSub subscription by full link, including project ID ([#2269](https://github.com/kedacore/keda/pull/2269))
- InfluxDB Scaler: add `unsafeSsl` parameter ([#2157](https://github.com/kedacore/keda/pull/2157)|[#2320](https://github.com/kedacore/keda/pull/2320))
- Metrics API Scaler: add Bearer auth ([#2028](https://github.com/kedacore/keda/pull/2028))
- MongoDB Scaler: add support to get connection data from Trigger Authorization ([#2115](https://github.com/kedacore/keda/pull/2115))
- MSSQL Scaler: add support to get connection data from Trigger Authorization ([#2112](https://github.com/kedacore/keda/pull/2112))
- MySQL Scaler: add support to get connection data from Trigger Authorization ([#2113](https://github.com/kedacore/keda/pull/2113))
- MySQL Scaler: don't expose connection string in `metricName` ([#2171](https://github.com/kedacore/keda/pull/2171))
- PostgreSQL Scaler: add support to get connection data from Trigger Authorization ([#2114](https://github.com/kedacore/keda/pull/2114))
- Prometheus Scaler: validating values length in Prometheus query response ([#2264](https://github.com/kedacore/keda/pull/2264))
- Prometheus Scaler: omit `serverAddress` from generated metric name ([#2099](https://github.com/kedacore/keda/pull/2099))
- RabbitMQ Scaler: anonymize the host in case of HTTP failure ([#2041](https://github.com/kedacore/keda/pull/2041))
- RabbitMQ Scaler: escape `queueName` and `vhostName` before use them in query string (bug fix) ([#2055](https://github.com/kedacore/keda/pull/2055))
- RabbitMQ Scaler: add custom http timeout ([#2086](https://github.com/kedacore/keda/pull/2086))
- RabbitMQ Scaler: add `pageSize` (using regex) ([#2162](https://github.com/kedacore/keda/pull/2162)|[#2319](https://github.com/kedacore/keda/pull/2319))
- Redis Scaler: upgrade library, add username and Sentinel support ([#2181](https://github.com/kedacore/keda/pull/2181))
- SeleniumGrid Scaler: add `unsafeSsl` parameter ([#2157](https://github.com/kedacore/keda/pull/2157))
- Stan Scaler: provide support for configuring authentication through TriggerAuthentication ([#2167](https://github.com/kedacore/keda/pull/2167))
- Allow setting `MaxConcurrentReconciles` for controllers ([#2272](https://github.com/kedacore/keda/pull/2272))
- Cleanup metric names inside scalers ([#2260](https://github.com/kedacore/keda/pull/2260))
- Drop support to `ValueMetricType` using cpu_memory_scaler ([#2218](https://github.com/kedacore/keda/issues/2218))
- Improve metric name creation to be unique using scaler index inside the scaler ([#2161](https://github.com/kedacore/keda/pull/2161))
- Improve error message if `IdleReplicaCount` are equal to `MinReplicaCount` to be the same as the check ([#2212](https://github.com/kedacore/keda/pull/2212))
- TriggerAuthentication/Vault: add support for HashiCorp Vault namespace (Vault Enterprise) ([#2085](https://github.com/kedacore/keda/pull/2085))

### Deprecations

- GCP PubSub Scaler: `subscriptionSize` is deprecated in favor of `mode` and `value` ([#2266](https://github.com/kedacore/keda/pull/2266))

### Breaking Changes

- TODO ([#XXX](https://github.com/kedacore/keda/pull/XXX))

### Other

- Ensure that `context.Context` values are properly passed down the stack ([#2202](https://github.com/kedacore/keda/pull/2202)|[#2249](https://github.com/kedacore/keda/pull/2249))
- Refactor AWS related scalers to reuse the AWS clients instead of creating a new one for every `GetMetrics` call ([#2255](https://github.com/kedacore/keda/pull/2255))
- Improve context handling in appropriate functionality in which we instantiate scalers ([#2267](https://github.com/kedacore/keda/pull/2267))
- Migrate to Kubebuilder v3 ([#2082](https://github.com/kedacore/keda/pull/2082))
    - API path has been changed: `github.com/kedacore/keda/v2/api/v1alpha1` -> `github.com/kedacore/keda/v2/apis/keda/v1alpha1`
- Use Patch to set FallbackCondition on ScaledObject.Status ([#2037](https://github.com/kedacore/keda/pull/2037))
- Bump Golang to 1.17.3 ([#2329](https://github.com/kedacore/keda/pull/2329))
- Add Makefile mockgen targets ([#2090](https://github.com/kedacore/keda/issues/2090)|[#2184](https://github.com/kedacore/keda/pull/2184))
- Add github action to run e2e command "on-demand" ([#2241](https://github.com/kedacore/keda/issues/2241))
- Add execution url in the pr-e2e triggering comment and fix problem related with not starting with ([#2306](https://github.com/kedacore/keda/issues/2306))

## v2.4.0

### New

- Add Solace PubSub+ Event Broker scaler ([#1945](https://github.com/kedacore/keda/pull/1945))
- Add Selenium Grid scaler ([#1971](https://github.com/kedacore/keda/pull/1971))
- Add Kubernetes Workload scaler ([#2010](https://github.com/kedacore/keda/pull/2010))
- Introduce fallback functionality ([#1872](https://github.com/kedacore/keda/issues/1872))
- Introduce Idle Replica Mode ([#1958](https://github.com/kedacore/keda/pull/1958))
- ScaledJob: Support pod conditions for pending job count calculation ([#1970](https://github.com/kedacore/keda/pull/1970)|[#2009](https://github.com/kedacore/keda/pull/2009))

### Improvements

- Optimize Kafka scaler by fetching all topic offsets using a single HTTP request ([#1956](https://github.com/kedacore/keda/pull/1956))
- Adding ability to specify Kafka Broker Version ([#1866](https://github.com/kedacore/keda/pull/1866))
- Support custom metric name in RabbitMQ scaler ([#1976](https://github.com/kedacore/keda/pull/1976))
- Support using regex to select the queues in RabbitMQ scaler ([#1957](https://github.com/kedacore/keda/pull/1957))
- Extend Azure Monitor scaler to support custom metrics ([#1883](https://github.com/kedacore/keda/pull/1883))
- Support non-public cloud environments in the Azure Service Bus scaler ([#1907](https://github.com/kedacore/keda/pull/1907))
- Support non-public cloud environments in the Azure Storage Queue and Azure Storage Blob scalers ([#1863](https://github.com/kedacore/keda/pull/1863))
- Adjusts InfluxDB scaler to support queries that return integers in addition to those that return floats ([#1977](https://github.com/kedacore/keda/pull/1977))
- Allow InfluxDB `authToken`, `serverURL`, and `organizationName` to be sourced from `(Cluster)TriggerAuthentication` ([#1904](https://github.com/kedacore/keda/pull/1904))
- IBM MQ scaler password handling fix ([#1939](https://github.com/kedacore/keda/pull/1939))
- Metrics APIServer: Add ratelimiting parameters to override client ([#1944](https://github.com/kedacore/keda/pull/1944))
- Fix READY and ACTIVE fields of ScaledJob to show status when we run `kubectl get sj` ([#1855](https://github.com/kedacore/keda/pull/1855))
- Show HashiCorp Vault Address when using `kubectl get ta` or `kubectl get cta` ([#1862](https://github.com/kedacore/keda/pull/1862))
- Don't panic when HashiCorp Vault path doesn't exist ([#1864](https://github.com/kedacore/keda/pull/1864))

### Breaking Changes

- Fix `keda-system-auth-delegator` ClusterRoleBinding name ([#1616](https://github.com/kedacore/keda/pull/1616). Upgrading may leave a stray ClusterRoleBinding with the old name `keda:system:auth-delegator` behind.

### Other

- Use `scaled[object/job].keda.sh/` prefix for KEDA related labels ([#2008](https://github.com/kedacore/keda/pull/2008))

## v2.3.0

### New

- Add Azure Pipelines Scaler ([#1706](https://github.com/kedacore/keda/pull/1706))
- Add OpenStack Metrics Scaler ([#1382](https://github.com/kedacore/keda/issues/1382))
- Added basic, tls and bearer authentication support to the Prometheus scaler [#1559](https://github.com/kedacore/keda/issues/1559)
- Add header Origin to Apache Artemis scaler [#1796](https://github.com/kedacore/keda/pull/1796)

### Improvements

- Azure Service Bus Scaler: Namespace from `connectionString` parameter is added to `metricName` due to uniqueness violation for clusters having more than one queue with the same name  ([#1755](https://github.com/kedacore/keda/issues/1755))
- Remove app.kubernetes.io/version label from label selectors ([#1696](https://github.com/kedacore/keda/pull/1696))
- Apache Kafka Scaler: Add `allowIdleConsumers` to the list of trigger parameters ([#1684](https://github.com/kedacore/keda/pull/1684))
- Fixed goroutine leaks in usage of timers ([#1704](https://github.com/kedacore/keda/pull/1704) | [#1739](https://github.com/kedacore/keda/pull/1739))
- Setting timeouts in the HTTP client used by the IBM MQ scaler ([#1758](https://github.com/kedacore/keda/pull/1758))
- Fix cleanup of removed triggers ([#1768](https://github.com/kedacore/keda/pull/1768))
- Eventhub Scaler: Add trigger parameter `checkpointStrategy` to support more language-specific checkpoints ([#1621](https://github.com/kedacore/keda/pull/1621))
- Fix Azure Blob scaler when using multiple triggers with the same `blobContainerName` and added a optional `metricName` field ([#1816](https://github.com/kedacore/keda/pull/1816))

### Breaking Changes

- None.

### Other

- Adding OpenStack Swift scaler end-to-end tests ([#1522](https://github.com/kedacore/keda/pull/1522))
- Pass deepCopy objects to the polling goroutines ([#1812](https://github.com/kedacore/keda/pull/1812))

## v2.2.0

### New

- Emit Kubernetes Events on KEDA events ([#1523](https://github.com/kedacore/keda/pull/1523) | [#1647](https://github.com/kedacore/keda/pull/1647))
- Support Quantities in Metrics API scaler ([#1667](https://github.com/kedacore/keda/issues/1667))
- Add Microsoft SQL Server (MSSQL) scaler ([#674](https://github.com/kedacore/keda/issues/674) | [docs](https://keda.sh/docs/2.2/scalers/mssql/))
- Add `publishRate` trigger to RabbitMQ scaler ([#1653](https://github.com/kedacore/keda/pull/1653))
- ScaledJob: support metadata labels in Job template ([#1686](https://github.com/kedacore/keda/pull/1686))

### Improvements

- Add `KEDA_HTTP_DEFAULT_TIMEOUT` support in Operator ([#1548](https://github.com/kedacore/keda/issues/1548))
- Removed `MIN field` for ScaledJob ([#1553](https://github.com/kedacore/keda/pull/1553))
- Add container port for Prometheus on Operator YAML ([#1562](https://github.com/kedacore/keda/pull/1562))
- Fix a memory leak in Kafka client and close push scalers ([#1565](https://github.com/kedacore/keda/issues/1565))
- Add 'Metadata' header to AAD podIdentity request ([#1566](https://github.com/kedacore/keda/issues/1566))
- KEDA should make sure generate correct labels for HPA ([#1630](https://github.com/kedacore/keda/issues/1630))
- Fix memory leak by checking triggers uniqueness properly ([#1640](https://github.com/kedacore/keda/pull/1640))
- Print correct ScaleTarget Kind in Events ([#1641](https://github.com/kedacore/keda/pull/1641))
- Fixed KEDA ClusterRoles to give permissions for ClusterTriggerAuthentications ([#1645](https://github.com/kedacore/keda/pull/1645))
- Make `swiftURL` parameter optional for the OpenStack Swift scaler ([#1652](https://github.com/kedacore/keda/pull/1652))
- Fix memory leak of `keda-metrics-apiserver` by setting a controller-runtime logger properly ([#1654](https://github.com/kedacore/keda/pull/1654))
- AWS SQS Scaler: Add Visible + NotVisible messages for scaling considerations ([#1664](https://github.com/kedacore/keda/pull/1664))
- Fixing behavior on ScaledJob with incorrect External Scaler ([#1672](https://github.com/kedacore/keda/pull/1672))

### Breaking Changes

- None.

### Other

- None.

## v2.1.0

### New

- Can use Pod Identity with Azure Event Hub scaler ([#994](https://github.com/kedacore/keda/issues/994))
- Introducing InfluxDB scaler ([#1239](https://github.com/kedacore/keda/issues/1239))
- Add Redis cluster support for Redis list and Redis streams scalers ([#1437](https://github.com/kedacore/keda/pull/1437))
- Global authentication credentials can be managed using `ClusterTriggerAuthentication` objects ([#1452](https://github.com/kedacore/keda/pull/1452))
- Introducing OpenStack Swift scaler ([#1342](https://github.com/kedacore/keda/issues/1342))
- Introducing MongoDB scaler ([#1467](https://github.com/kedacore/keda/pull/1467))

### Improvements

- Support add ScaledJob's label to its job ([#1311](https://github.com/kedacore/keda/issues/1311))
- Bug fix in aws_iam_authorization to utilize correct secret from env key name ([#1332](https://github.com/kedacore/keda/pull/1332))
- Add metricName field to postgres scaler and auto generate if not defined ([#1381](https://github.com/kedacore/keda/pull/1381))
- Mask password in postgres scaler auto generated metricName ([#1381](https://github.com/kedacore/keda/pull/1381))
- Bug fix for pending jobs in ScaledJob's accurateScalingStrategy ([#1323](https://github.com/kedacore/keda/issues/1323))
- Fix memory leak because of unclosed scalers ([#1413](https://github.com/kedacore/keda/issues/1413))
- Override the vhost on a RabbitMQ scaler via `vhostName` in the metadata ([#1451](https://github.com/kedacore/keda/pull/1451))
- Optimize Kafka scaler's `getLagForPartition` function ([#1464](https://github.com/kedacore/keda/pull/1464))
- Reduce unnecessary /scale requests from ScaledObject controller ([#1453](https://github.com/kedacore/keda/pull/1453))
- Add support for the `WATCH_NAMESPACE` environment variable to the operator ([#1474](https://github.com/kedacore/keda/pull/1474))
- Automatically determine the RabbitMQ protocol when possible, and support setting the protocl via TriggerAuthentication ([#1459](https://github.com/kedacore/keda/pull/1459), [#1483](https://github.com/kedacore/keda/pull/1483))
- Improve performance when fetching pod information ([#1457](https://github.com/kedacore/keda/pull/1457))
- Improve performance when fetching current scaling information on Deployments ([#1458](https://github.com/kedacore/keda/pull/1458))
- Improve error reporting in prometheus scaler ([#1497](https://github.com/kedacore/keda/pull/1497))
- Check that metricNames are unique in ScaledObject ([#1390](https://github.com/kedacore/keda/pull/1390))
- Serve OpenAPI spec from KEDA Metrics Apiserver ([#1512](https://github.com/kedacore/keda/pull/1512))
- Support metrics with multiple dimensions and configurable metricValues on AWS Cloudwatch Scaler ([#1230](https://github.com/kedacore/keda/issues/1230))
- Show `MIN/MAX` replica counts when using `kubectl get scaledobject/scaledjob` ([#1534](https://github.com/kedacore/keda/pull/1534))
- Fix unnecessary HPA updates when Resource based Trigger is used ([#1541](https://github.com/kedacore/keda/pull/1541))

### Breaking Changes

None.

### Other

- Bump go module version to v2 ([#1324](https://github.com/kedacore/keda/pull/1324))

## v2.0.0

### New

- KEDA uses a dedicated [HTTP client](https://pkg.go.dev/net/http#Client), connection pool, and (optional) TLS certificate for each configured scaler
- KEDA scales any CustomResource that implements Scale subresource ([#703](https://github.com/kedacore/keda/issues/703))
- Provide KEDA go-client ([#494](https://github.com/kedacore/keda/issues/494))
- Define KEDA readiness and liveness probes ([#788](https://github.com/kedacore/keda/issues/788))
- KEDA Support for configurable scaling behavior in HPA v2beta2 ([#802](https://github.com/kedacore/keda/issues/802))
- Add External Push scaler ([#820](https://github.com/kedacore/keda/issues/820) | [docs](https://keda.sh/docs/2.0/scalers/external-push/))
- Managed Identity support for Azure Monitor scaler ([#936](https://github.com/kedacore/keda/issues/936))
- Add support for multiple triggers on ScaledObject ([#476](https://github.com/kedacore/keda/issues/476))
- Add consumer offset reset policy option to Kafka scaler ([#925](https://github.com/kedacore/keda/pull/925))
- Add option to restore to original replica count after ScaledObject's deletion ([#219](https://github.com/kedacore/keda-docs/pull/219))
- Add Prometheus metrics for KEDA Metrics API Server ([#823](https://github.com/kedacore/keda/issues/823) | [docs](https://keda.sh/docs/2.0/operate/#prometheus-exporter-metrics))
- Add support for multiple redis list types in redis list scaler ([#1006](https://github.com/kedacore/keda/pull/1006)) | [docs](https://keda.sh/docs/2.0/scalers/redis-lists/))
- Introduce Azure Log Analytics scaler ([#1061](https://github.com/kedacore/keda/issues/1061)) | [docs](https://keda.sh/docs/2.0/scalers/azure-log-analytics/))
- Add Metrics API Scaler ([#1026](https://github.com/kedacore/keda/pull/1026))
- Add cpu/memory Scaler ([#1215](https://github.com/kedacore/keda/pull/1215))
- Add Scaling Strategy for ScaledJob ([#1227](https://github.com/kedacore/keda/pull/1227))
- Add IBM MQ Scaler ([#1253](https://github.com/kedacore/keda/issues/1253))

### Improvements

- Move from autoscaling `v2beta1` to `v2beta2` for HPA ([#721](https://github.com/kedacore/keda/issues/721))
- Introduce shortnames for CRDs ([#774](https://github.com/kedacore/keda/issues/774))
- Improve `kubectl get scaledobject` to show related trigger authentication ([#777](https://github.com/kedacore/keda/issues/777))
- Improve `kubectl get triggerauthentication` to show information about configured parameters ([#778](https://github.com/kedacore/keda/issues/778))
- Added ScaledObject Status Conditions to display status of scaling ([#750](https://github.com/kedacore/keda/pull/750))
- Added optional authentication parameters for the Redis Scaler ([#962](https://github.com/kedacore/keda/pull/962))
- Improved GCP PubSub Scaler performance by closing the client correctly ([#1087](https://github.com/kedacore/keda/pull/1087))
- Added support for Trigger Authentication for GCP PubSub scaler ([#1291](https://github.com/kedacore/keda/pull/1291))

### Breaking Changes

- Change `apiGroup` from `keda.k8s.io` to `keda.sh` ([#552](https://github.com/kedacore/keda/issues/552))
- Introduce a separate ScaledObject and ScaledJob([#653](https://github.com/kedacore/keda/issues/653))
- Remove `New()` and `Close()` from the interface of `service ExternalScaler` in `externalscaler.proto`.
- Removed deprecated brokerList for Kafka scaler ([#882](https://github.com/kedacore/keda/pull/882))
- All scalers metadata that is resolved from the scaleTarget environment have suffix `FromEnv` added. e.g: `connection` -> `connectionFromEnv`
- Kafka: split metadata and config for SASL and TLS ([#1074](https://github.com/kedacore/keda/pull/1074))
- Service Bus: `queueLength` is now called `messageCount` ([#1109](https://github.com/kedacore/keda/issues/1109))
- Use `host` instead of `apiHost` in `rabbitmq` scaler. Add `protocol` in trigger spec to specify which protocol should be used ([#1115](https://github.com/kedacore/keda/pull/1115))
- CRDs are using `apiextensions.k8s.io/v1` apiVersion ([#1202](https://github.com/kedacore/keda/pull/1202))

### Other
- Change API optional structs to pointers to conform with k8s guide ([#1170](https://github.com/kedacore/keda/issues/1170))
- Update Operator SDK and k8s deps ([#1007](https://github.com/kedacore/keda/pull/1007),[#870](https://github.com/kedacore/keda/issues/870),[#1180](https://github.com/kedacore/keda/pull/1180))
- Change Metrics Server image name from `keda-metrics-adapter` to `keda-metrics-apiserver` ([#1105](https://github.com/kedacore/keda/issues/1105))
- Add Argo Rollouts e2e test ([#1234](https://github.com/kedacore/keda/issues/1234))

## v1.5.0

Learn more about our release in [our milestone](https://github.com/kedacore/keda/milestone/12).

### New

- **Scalers**
    - Introduce Active MQ Artemis scaler ([Docs](https://keda.sh/docs/1.5/scalers/artemis/))
    - Introduce Redis Streams scaler ([Docs](https://keda.sh/docs/1.5/scalers/redis-streams/) | [Details](https://github.com/kedacore/keda/issues/746))
    - Introduce Cron scaler ([Docs](https://keda.sh/docs/1.5/scalers/cron/) | [Details](https://github.com/kedacore/keda/issues/812))
- **Secret Providers**
    - Introduce HashiCorp Vault secret provider ([Docs](https://keda.sh/docs/1.5/concepts/authentication/#hashicorp-vault-secrets) | [Details](https://github.com/kedacore/keda/issues/673))
- **Other**
    - Introduction of `nodeSelector` in raw YAML deployment specifications ([Details](https://github.com/kedacore/keda/pull/856))

### Improvements

- Improved message count determination when using `includeUnacked` in RabbitMQ scaler ([Details](https://github.com/kedacore/keda/pull/781))
- Fix for blank path without trailing slash in RabbitMQ scaler ([Details](https://github.com/kedacore/keda/issues/790))
- Improved parsing of connection strings to support `BlobEndpoint`, `QueueEndpoint`, `TableEndpoint` & `FileEndpoint` segments ([Details](https://github.com/kedacore/keda/issues/821))
- Support scaling when no storage checkpoint exists in Azure Event Hubs scaler ([Details](https://github.com/kedacore/keda/issues/797))
- GCP Pub Scaler should not panic on invalid credentials ([Details](https://github.com/kedacore/keda/issues/616))
- Make `queueLength` optional in RabbitMQ scaler ([Details](https://github.com/kedacore/keda/issues/880))

### Breaking Changes

None.

### Other

None.

## v1.4.1

### New

None

### Improvements

- Fix for scale-to-zero for Prometheus scaler no longer working ([#770](https://github.com/kedacore/keda/issues/770))
- Fix for passing default VHost for Rabbit MQ scaler no longer working ([#770](https://github.com/kedacore/keda/issues/768))
- Provide capability to define time encoding for operator ([#766](https://github.com/kedacore/keda/pull/766))

### Breaking Changes

None.

### Other

- Print version of metric adapter in logs ([#770](https://github.com/kedacore/keda/issues/748))

## v1.4.0

### New

- Extend RabbitMQ scaler to support count unacked messages([#700](https://github.com/kedacore/keda/pull/700))

### Improvements

- Fix scalers leaking ([#684](https://github.com/kedacore/keda/pull/684))
- Provide installation YAML package as release artifact ([#740](https://github.com/kedacore/keda/pull/740))
- Improve Azure Monitor scaler to handle queries without metrics ([#680](https://github.com/kedacore/keda/pull/680))
- Authenticate to AWS with dedicated role without AssumeRole permissions ([#656](https://github.com/kedacore/keda/pull/656))
- KEDA now respects label restrictions on Horizontal Pod Autoscaler to have max 63 chars ([#707](https://github.com/kedacore/keda/pull/707))
- KEDA will automatically assign `deploymentName` label if it was not defined in `ScaledObject` ([#709](https://github.com/kedacore/keda/pull/709))

### Breaking Changes

None.

### Other

- Adding label for metrics service selection ([#745](https://github.com/kedacore/keda/pull/745))
- Filter returned metrics from api server based on queried name ([#732](https://github.com/kedacore/keda/pull/732))
- Add redis host and port parameter to the scaler with tests ([#719](https://github.com/kedacore/keda/pull/719))
- Remove go micro version ([#718](https://github.com/kedacore/keda/pull/718))
- Update zero result return to be non-error inducing ([#695](https://github.com/kedacore/keda/pull/695))
- Return if kafka offset response is nil ([#689](https://github.com/kedacore/keda/pull/689))
- Fix typos in MySQL scaler ([#683](https://github.com/kedacore/keda/pull/683))
- Update README to mention CNCF ([#682](https://github.com/kedacore/keda/pull/682))

## v1.3.0

### New

- Add Azure monitor scaler ([#584](https://github.com/kedacore/keda/pull/584))
- Introduce changelog ([#664](https://github.com/kedacore/keda/pull/664))
- Introduce support for AWS pod identity ([#499](https://github.com/kedacore/keda/pull/499))

### Improvements

- Make targetQueryValue configurable in postgreSQL scaler ([#643](https://github.com/kedacore/keda/pull/643))
- Removed the need for deploymentName label ([#644](https://github.com/kedacore/keda/pull/644))
- Adding Kubernetes recommended labels to resources ([#596](https://github.com/kedacore/keda/pull/596))

### Breaking Changes

None.

### Other

- Updating license to Apache per CNCF donation ([#661](https://github.com/kedacore/keda/pull/661))

## v1.2.0

### New

- Introduce new Postgres scaler ([#553](https://github.com/kedacore/keda/issues/553))
- Introduce new MySQL scaler ([#564](https://github.com/kedacore/keda/issues/564))
- Provide SASL_SSL Plain authentication for Kafka trigger scalar to work with Event Hubs ([#585](https://github.com/kedacore/keda/issues/585))

### Improvements

- TLS parameter to Redis-scaler ([#540](https://github.com/kedacore/keda/issues/540))
- Redis db index option ([#577](https://github.com/kedacore/keda/issues/577))
- Optional param for ConfigMaps and Secrets ([#562](https://github.com/kedacore/keda/issues/562))
- Remove manually adding sslmode to connection string ([#558](https://github.com/kedacore/keda/issues/558))
- ScaledObject.Status update should handle stale resource ([#582](https://github.com/kedacore/keda/issues/582))
- Improve reconcile loop ([#581](https://github.com/kedacore/keda/issues/581))
- Address naming changes for postgresql scaler ([#593](https://github.com/kedacore/keda/issues/593))

### Breaking Changes

None.

### Other

- Move Metrics adapter into the separate Deployment ([#506](https://github.com/kedacore/keda/issues/506))
- Fix gopls location ([#574](https://github.com/kedacore/keda/issues/574))
- Add instructions on local development and debugging ([#583](https://github.com/kedacore/keda/issues/583))
- Add a checkenv target ([#600](https://github.com/kedacore/keda/issues/600))
- Mentioning problem with checksum mismatch error ([#605](https://github.com/kedacore/keda/issues/605))

## v1.1.0

### New

- Introduce new Huawei Cloud CloudEye scaler ([#478](https://github.com/kedacore/keda/issues/478))
- Introduce new kinesis stream scaler ([#526](https://github.com/kedacore/keda/issues/526))
- Introduce new Azure blob scaler ([#514](https://github.com/kedacore/keda/issues/514))
- Support for SASL authentication for Kafka scaler ([#486](https://github.com/kedacore/keda/issues/486))

### Improvements

- Event Hub scalar expansion to work with Java and C# applications ([#517](https://github.com/kedacore/keda/issues/517))
- Escape Prometheus querystring ([#521](https://github.com/kedacore/keda/issues/521))
- Change how number of pending messages is calculated and add more error handling. ([#533](https://github.com/kedacore/keda/issues/533))
- Service bus scaler pod identity fix ([#534](https://github.com/kedacore/keda/issues/534))
- Eventhub scalar fix ([#537](https://github.com/kedacore/keda/issues/537))
- Kafka scaler fix for SASL plaintext auth ([#544](https://github.com/kedacore/keda/issues/544))

### Breaking Changes

None.

### Other

- ScaledObject Status clean up ([#466](https://github.com/kedacore/keda/issues/466))
- Add default log level for operator ([#468](https://github.com/kedacore/keda/issues/468))
- Ensure get the metrics that have been aggregated ([#509](https://github.com/kedacore/keda/issues/509))
- Scale from zero when minReplicaCount is > 0 ([#524](https://github.com/kedacore/keda/issues/524))
- Total running Jobs must not exceed maxScale - Running jobs ([#528](https://github.com/kedacore/keda/issues/528))
- Check deploymentName definition in ScaledObject ([#532](https://github.com/kedacore/keda/issues/532))

## v1.0.0

### New

- Many more scalers added
- Scaler extensibility (run scalers in a different container and communicate with KEDA via gRPC)
- TriggerAuthentication and Pod Identity for identity based auth that can be shared across deployments
- Schedule jobs on events in addition scaling out deployments

### Improvements

- Additional tests and automation through GitHub Actions

### Breaking Changes

- RabbitMQ `host` property now must resolve from a secret ([#347](https://github.com/kedacore/keda/issues/347))

### Other

None.
