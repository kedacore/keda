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

### New

- **General**: Expand Prometheus metric with label "ScalerName" to distinguish different triggers. The scaleName is defined per Trigger.Name ([#3588](https://github.com/kedacore/keda/issues/3588)
- **General:** Introduce new Loki Scaler ([#3699](https://github.com/kedacore/keda/issues/3699))
- **General**: Add ratelimitting parameters to keda manager to allow override of client defaults ([#3730](https://github.com/kedacore/keda/issues/2920))
- **General**: Provide Prometheus metric with indication of total number of triggers per trigger type in `ScaledJob`/`ScaledObject`. ([#3663](https://github.com/kedacore/keda/issues/3663))
- **AWS Scalers**: Add setting AWS endpoint url. ([#3337](https://github.com/kedacore/keda/issues/3337))
- **Azure Service Bus Scaler**: Add support for Shared Access Signature (SAS) tokens for authentication. ([#2920](https://github.com/kedacore/keda/issues/2920))
- **Azure Service Bus Scaler:** Support regex usage in queueName / subscriptionName parameters. ([#1624](https://github.com/kedacore/keda/issues/1624))
- **Selenium Grid Scaler:** Allow setting url trigger parameter from TriggerAuthentication/ClusterTriggerAuthentication ([#3752](https://github.com/kedacore/keda/pull/3752))

### Improvements

- **General:** Add explicit seccompProfile type to securityContext config ([#3561](https://github.com/kedacore/keda/issues/3561))
- **General:** Add `Min` column to ScaledJob visualization ([#3689](https://github.com/kedacore/keda/issues/3689))
- **General:** Add support to use pod identities for authentication in Azure Key Vault ([#3813](https://github.com/kedacore/keda/issues/3813)
- **Apache Kafka Scaler:** SASL/OAuthbearer Implementation ([#3681](https://github.com/kedacore/keda/issues/3681))
- **Azure AD Pod Identity Authentication:** Improve error messages to emphasize problems around the integration with aad-pod-identity itself ([#3610](https://github.com/kedacore/keda/issues/3610))
- **Azure Pipelines Scaler:** Improved speed of profiling large set of Job Requests from Azure Pipelines ([#3702](https://github.com/kedacore/keda/issues/3702))
- **GCP Storage Scaler:** Add prefix and delimiter support ([#3756](https://github.com/kedacore/keda/issues/3756))
- **Prometheus Scaler:** Introduce skipping of certificate check for unsigned certs ([#2310](https://github.com/kedacore/keda/issues/2310))
- **Event Hubs Scaler:** Support Azure Active Direcotry Pod & Workload Identity for Storage Blobs ([#3569](https://github.com/kedacore/keda/issues/3569))
- **Metrics API Scaler:** Add unsafeSsl paramater to skip certificate validation when connecting over HTTPS ([#3728](https://github.com/kedacore/keda/discussions/3728))

### Fixes

- **General:** Provide patch for CVE-2022-3172 vulnerability ([#3690](https://github.com/kedacore/keda/issues/3690))
- **General:** Respect optional parameter inside envs for ScaledJobs ([#3568](https://github.com/kedacore/keda/issues/3568))
- **Azure Blob Scaler** Store forgotten logger ([#3811](https://github.com/kedacore/keda/issues/3811))
- **Prometheus Scaler:** Treat Inf the same as Null result ([#3644](https://github.com/kedacore/keda/issues/3644))

### Deprecations

- TODO ([#XXX](https://github.com/kedacore/keda/issue/XXX))

### Breaking Changes

- **General:** Change API version of HPA from `autoscaling/v2beta2` to `autoscaling/v2` ([#2462](https://github.com/kedacore/keda/issues/2462))

### Other

- **General**: Bump Golang to 1.18.6 ([#3205](https://github.com/kedacore/keda/issues/3205))
- **General**: Bump `github.com/Azure/azure-event-hubs-go/v3` ([#2986](https://github.com/kedacore/keda/issues/2986))
- **General**: Migrate from `azure-service-bus-go` to `azservicebus` ([#3394](https://github.com/kedacore/keda/issues/3394))
- **Azure EventHub**: Add e2e tests ([#2792](https://github.com/kedacore/keda/issues/2792))

## v2.8.1

### New

None.

### Improvements

- **Datadog Scaler:** Support multi-query metrics, and aggregation ([#3423](https://github.com/kedacore/keda/issues/3423))

### Fixes

- **General:** Metrics endpoint returns correct HPA values ([#3554](https://github.com/kedacore/keda/issues/3554))
- **Datadog Scaler:** Fix: panic in datadog scaler ([#3448](https://github.com/kedacore/keda/issues/3448))
- **RabbitMQ Scaler:** Parse vhost correctly if it's provided in the host url ([#3602](https://github.com/kedacore/keda/issues/3602))

### Deprecations

None.

### Breaking Changes

None.

### Other

- **General:** Execute trivy scan (on PRs) only if there are changes in deps ([#3540](https://github.com/kedacore/keda/issues/3540))
- **General:** Use re-usable workflows for GitHub Actions ([#2569](https://github.com/kedacore/keda/issues/2569))

## v2.8.0

### New

- **General:** Introduce new AWS DynamoDB Streams Scaler ([#3124](https://github.com/kedacore/keda/issues/3124))
- **General:** Introduce new NATS JetStream scaler ([#2391](https://github.com/kedacore/keda/issues/2391))
- **General:** Introduce `activationThreshold`/`minMetricValue` for all scalers ([#2800](https://github.com/kedacore/keda/issues/2800))
- **General:** Support for `minReplicaCount` in ScaledJob ([#3426](https://github.com/kedacore/keda/issues/3426))
- **General:** Support to customize HPA name ([#3057](https://github.com/kedacore/keda/issues/3057))
- **General:** Make propagation policy for ScaledJob rollout configurable ([#2910](https://github.com/kedacore/keda/issues/2910))
- **General:** Support for Azure AD Workload Identity as a pod identity provider. ([#2487](https://github.com/kedacore/keda/issues/2487)|[#2656](https://github.com/kedacore/keda/issues/2656))
- **General:** Support for permission segregation when using Azure AD Pod / Workload Identity. ([#2656](https://github.com/kedacore/keda/issues/2656))
- **AWS SQS Queue Scaler:** Support for scaling to include in-flight messages. ([#3133](https://github.com/kedacore/keda/issues/3133))
- **Azure Pipelines Scaler:** Support for Azure Pipelines to support demands (capabilities) ([#2328](https://github.com/kedacore/keda/issues/2328))
- **CPU Scaler:** Support for targeting specific container in a pod ([#1378](https://github.com/kedacore/keda/issues/1378))
- **GCP Stackdriver Scaler:** Added aggregation parameters ([#3008](https://github.com/kedacore/keda/issues/3008))
- **Kafka Scaler:** Support of passphrase encrypted PKCS #\8 private key ([3449](https://github.com/kedacore/keda/issues/3449))
- **Memory Scaler:** Support for targeting specific container in a pod ([#1378](https://github.com/kedacore/keda/issues/1378))
- **Prometheus Scaler:** Add `ignoreNullValues` to return error when prometheus return null in values ([#3065](https://github.com/kedacore/keda/issues/3065))

### Improvements

- **General:** Add settings for configuring leader election ([#2836](https://github.com/kedacore/keda/issues/2836))
- **General:** `external` extension reduces connection establishment with long links ([#3193](https://github.com/kedacore/keda/issues/3193))
- **General:** Reference ScaledObject's/ScaledJob's name in the scalers log ([3419](https://github.com/kedacore/keda/issues/3419))
- **General:** Use `mili` scale for the returned metrics ([#3135](https://github.com/kedacore/keda/issues/3135))
- **General:** Use more readable timestamps in KEDA Operator logs ([#3066](https://github.com/kedacore/keda/issues/3066))
- **Kafka Scaler:** Handle Sarama errors properly ([#3056](https://github.com/kedacore/keda/issues/3056))

### Fixes

- **General:** Provide patch for CVE-2022-27191 vulnerability ([#3378](https://github.com/kedacore/keda/issues/3378))
- **General:** Refactor adapter startup to ensure proper log initilization. ([2316](https://github.com/kedacore/keda/issues/2316))
- **General:** Scaleobject ready condition 'False/Unknow' to 'True' requeue ([#3096](https://github.com/kedacore/keda/issues/3096))
- **General:** Use `go install` in the Makefile for downloading dependencies ([#2916](https://github.com/kedacore/keda/issues/2916))
- **General:** Use metricName from GetMetricsSpec in ScaledJobs instead of `queueLength` ([#3032](https://github.com/kedacore/keda/issues/3032))
- **ActiveMQ Scaler:** KEDA doesn't respect restAPITemplate ([#3188](https://github.com/kedacore/keda/issues/3188))
- **Azure Eventhub Scaler:** KEDA operator crashes on nil memory panic if the eventhub connectionstring for Azure Eventhub Scaler contains an invalid character ([#3082](https://github.com/kedacore/keda/issues/3082))
- **Azure Pipelines Scaler:** Fix issue with Azure Pipelines wrong PAT Auth. ([#3159](https://github.com/kedacore/keda/issues/3159))
- **Datadog Scaler:** Ensure that returns the same element that has been checked ([#3448](https://github.com/kedacore/keda/issues/3448))
- **Kafka Scaler:** Check `lagThreshold` is a positive number ([#3366](https://github.com/kedacore/keda/issues/3366))
- **Selenium Grid Scaler:** Fix bug where edge active sessions not being properly counted ([#2709](https://github.com/kedacore/keda/issues/2709))
- **Selenium Grid Scaler:** Fix bug where Max Sessions was not working correctly ([#3061](https://github.com/kedacore/keda/issues/3061))

### Deprecations

- **ScaledJob**: `rolloutStrategy` is deprecated in favor of `rollout.strategy` ([#2910](https://github.com/kedacore/keda/issues/2910))

### Breaking Changes

None.

### Other

- **General:** Migrate e2e test to Go. ([2737](https://github.com/kedacore/keda/issues/2737))
- **General**: Bump Golang to 1.17.13 and deps ([#3447](https://github.com/kedacore/keda/issues/3447))
- **General:** Fix devcontainer on ARM64 Arch. ([3084](https://github.com/kedacore/keda/issues/3084))
- **General:** Improve error message in resolving ServiceAccount for AWS EKS PodIdentity ([3142](https://github.com/kedacore/keda/issues/3142))
- **General:** Improve e2e on PR process through comments. ([3004](https://github.com/kedacore/keda/issues/3004))
- **General:** Split e2e test by functionality. ([#3270](https://github.com/kedacore/keda/issues/3270))
- **General:** Unify the used tooling on different workflows and arch. ([3092](https://github.com/kedacore/keda/issues/3092))
- **General:** Use Github's Checks API for e2e tests on PR. ([2567](https://github.com/kedacore/keda/issues/2567))

## v2.7.1

### Improvements

- **General**: Don't hardcode UIDs in securityContext ([#3012](https://github.com/kedacore/keda/issues/3012))

### Other

- **General**: Fix CVE-2022-21221 in `github.com/valyala/fasthttp` ([#2775](https://github.com/kedacore/keda/issues/2775))
- **General**: Bump Golang to 1.17.9 ([#3016](https://github.com/kedacore/keda/issues/3016))
- **General**: Fix autoscaling behaviour while paused. ([#3009](https://github.com/kedacore/keda/issues/3009))

## v2.7.0

### New

- **General:** Introduce annotation `"autoscaling.keda.sh/paused-replicas"` for ScaledObjects to pause scaling at a fixed replica count. ([#944](https://github.com/kedacore/keda/issues/944))
- **General:** Introduce ARM-based container image for KEDA ([#2263](https://github.com/kedacore/keda/issues/2263)|[#2262](https://github.com/kedacore/keda/issues/2262))
- **General:** Introduce new AWS DynamoDB Scaler ([#2486](https://github.com/kedacore/keda/issues/2482))
- **General:** Introduce new Azure Data Explorer Scaler ([#1488](https://github.com/kedacore/keda/issues/1488)|[#2734](https://github.com/kedacore/keda/issues/2734))
- **General:** Introduce new GCP Stackdriver Scaler ([#2661](https://github.com/kedacore/keda/issues/2661))
- **General:** Introduce new GCP Storage Scaler ([#2628](https://github.com/kedacore/keda/issues/2628))
- **General:** Provide support for authentication via Azure Key Vault ([#900](https://github.com/kedacore/keda/issues/900)|[#2733](https://github.com/kedacore/keda/issues/2733))
- **General**: Support for `ValueMetricType` in `ScaledObject` for all scalers except CPU/Memory ([#2030](https://github.com/kedacore/keda/issues/2030))

### Improvements

- **General:** Bump dependencies versions ([#2978](https://github.com/kedacore/keda/issues/2978))
- **General:** Properly handle `restoreToOriginalReplicaCount` if `ScaleTarget` is missing ([#2872](https://github.com/kedacore/keda/issues/2872))
- **General:** Support for running KEDA secure-by-default as non-root ([#2933](https://github.com/kedacore/keda/issues/2933))
- **General:** Synchronize HPA annotations from ScaledObject ([#2659](https://github.com/kedacore/keda/pull/2659))
- **General:** Updated HTTPClient to be proxy-aware, if available, from environment variables. ([#2577](https://github.com/kedacore/keda/issues/2577))
- **General:** Using manager client in KEDA Metrics Server to avoid flush request to Kubernetes Apiserver([2914](https://github.com/kedacore/keda/issues/2914))
- **ActiveMQ Scaler:** Add CorsHeader information to ActiveMQ Scaler ([#2884](https://github.com/kedacore/keda/issues/2884))
- **AWS CloudWatch:** Add support to use expressions([#2998](https://github.com/kedacore/keda/issues/2998))
- **Azure Application Insights Scaler:** Provide support for non-public clouds ([#2735](https://github.com/kedacore/keda/issues/2735))
- **Azure Blob Storage Scaler:** Add optional parameters for counting blobs recursively ([#1789](https://github.com/kedacore/keda/issues/1789))
- **Azure Event Hub Scaler:** Improve logging when blob container not found ([#2363](https://github.com/kedacore/keda/issues/2363))
- **Azure Event Hub Scaler:** Provide support for non-public clouds ([#1915](https://github.com/kedacore/keda/issues/1915))
- **Azure Log Analytics Scaler:** Provide support for non-public clouds ([#1916](https://github.com/kedacore/keda/issues/1916))
- **Azure Monitor Scaler:** Provide support for non-public clouds ([#1917](https://github.com/kedacore/keda/issues/1917))
- **Azure Queue:** Don't call Azure queue GetProperties API unnecessarily ([#2613](https://github.com/kedacore/keda/pull/2613))
- **Datadog Scaler:** Validate query to contain `{` to prevent panic on invalid query ([#2625](https://github.com/kedacore/keda/issues/2625))
- **Datadog Scaler:** Several improvements, including a new optional parameter `metricUnavailableValue` to fill data when no Datadog metric was returned ([#2657](https://github.com/kedacore/keda/issues/2657))
- **Datadog Scaler:** Rely on Datadog API to validate the query ([2761](https://github.com/kedacore/keda/issues/2761))
- **Graphite Scaler**  Use the latest non-null datapoint returned by query. ([#2625](https://github.com/kedacore/keda/issues/2944))
- **Kafka Scaler:** Make "disable" a valid value for tls auth parameter ([#2608](https://github.com/kedacore/keda/issues/2608))
- **Kafka Scaler:** New `scaleToZeroOnInvalidOffset` to control behavior when partitions have an invalid offset ([#2033](https://github.com/kedacore/keda/issues/2033)[#2612](https://github.com/kedacore/keda/issues/2612))
- **Metric API Scaler:** Improve error handling on not-ok response ([#2317](https://github.com/kedacore/keda/issues/2317))
- **New Relic Scaler:** Support to get account value from authentication resources. ([#2883](https://github.com/kedacore/keda/issues/2883))
- **Prometheus Scaler:** Check and properly inform user that `threshold` is not set ([#2793](https://github.com/kedacore/keda/issues/2793))
- **Prometheus Scaler:** Support for `X-Scope-OrgID` header ([#2667](https://github.com/kedacore/keda/issues/2667))
- **RabbitMQ Scaler:** Include `vhost` for RabbitMQ when retrieving queue info with `useRegex` ([#2498](https://github.com/kedacore/keda/issues/2498))
- **Selenium Grid Scaler:** Consider `maxSession` grid info when scaling. ([#2618](https://github.com/kedacore/keda/issues/2618))

### Deprecations

- **CPU, Memory, Datadog Scalers**: `metadata.type` is deprecated in favor of the global `metricType` ([#2030](https://github.com/kedacore/keda/issues/2030))

### Breaking Changes

None.

### Other

- **General:** Clean go.mod to fix golangci-lint ([#2783](https://github.com/kedacore/keda/issues/2783))
- **General:** Consistent file naming in `pkg/scalers/` ([#2806](https://github.com/kedacore/keda/issues/2806))
- **General:** Fix mismatched errors for updating HPA ([#2719](https://github.com/kedacore/keda/issues/2719))
- **General:** Improve e2e tests reliability ([#2580](https://github.com/kedacore/keda/issues/2580))
- **General:** Improve e2e tests to always cleanup resources in cluster ([#2584](https://github.com/kedacore/keda/issues/2584))
- **General:** Internally represent value and threshold as int64 ([#2790](https://github.com/kedacore/keda/issues/2790))
- **General:** Refactor active directory endpoint parsing for Azure scalers. ([#2853](https://github.com/kedacore/keda/pull/2853))
- **AWS CloudWatch:** Adding e2e test ([#1525](https://github.com/kedacore/keda/issues/1525))
- **AWS DynamoDB:** Setup AWS DynamoDB test account ([#2803](https://github.com/kedacore/keda/issues/2803))
- **AWS Kinesis Stream:** Adding e2e test ([#1526](https://github.com/kedacore/keda/issues/1526))
- **AWS SQS Queue:** Adding e2e test ([#1527](https://github.com/kedacore/keda/issues/1527))
- **Azure Data Explorer:** Adding e2e test ([#2841](https://github.com/kedacore/keda/issues/2841))
- **Azure Data Explorer:** Replace deprecated function `iter.Next()` in favour of `iter.NextRowOrError()` ([#2989](https://github.com/kedacore/keda/issues/2989))
- **Azure Service Bus:** Adding e2e test ([#2731](https://github.com/kedacore/keda/issues/2731)|[#2732](https://github.com/kedacore/keda/issues/2732))
- **External Scaler:** Adding e2e test. ([#2697](https://github.com/kedacore/keda/issues/2697))
- **External Scaler:** Fix issue with internal KEDA core prefix being passed to external scaler. ([#2640](https://github.com/kedacore/keda/issues/2640))
- **GCP Pubsub Scaler:** Adding e2e test ([#1528](https://github.com/kedacore/keda/issues/1528))
- **Hashicorp Vault Secret Provider:** Adding e2e test ([#2842](https://github.com/kedacore/keda/issues/2842))
- **Memory Scaler:** Adding e2e test ([#2220](https://github.com/kedacore/keda/issues/2220))
- **Selenium Grid Scaler:** Adding e2e test ([#2791](https://github.com/kedacore/keda/issues/2791))

## v2.6.1

### Improvements

- **General**: Fix generation of metric names if any of ScaledObject's triggers is unavailable ([#2592](https://github.com/kedacore/keda/issues/2592))
- **General**: Fix logging in KEDA operator and properly set `ScaledObject.Status` in case there is a problem in a ScaledObject's trigger ([#2603](https://github.com/kedacore/keda/issues/2603))

### Other

- **General:** Fix failing tests based on the scale to zero bug ([#2603](https://github.com/kedacore/keda/issues/2603))

## v2.6.0

### New

- Add ActiveMQ Scaler ([#2305](https://github.com/kedacore/keda/pull/2305))
- Add Azure Application Insights Scaler ([2506](https://github.com/kedacore/keda/pull/2506))
- Add New Datadog Scaler ([#2354](https://github.com/kedacore/keda/pull/2354))
- Add New Relic Scaler ([#2387](https://github.com/kedacore/keda/pull/2387))
- Add PredictKube Scaler ([#2418](https://github.com/kedacore/keda/pull/2418))

### Improvements

- **General:** Delete the cache entry when a ScaledObject is deleted ([#2564](https://github.com/kedacore/keda/pull/2564))
- **General:** Fail fast on `buildScalers` when not able to resolve a secret that a deployment is relying on ([#2394](https://github.com/kedacore/keda/pull/2394))
- **General:** `keda-operator` Cluster Role: add `list` and `watch` access to service accounts ([#2406](https://github.com/kedacore/keda/pull/2406))|([#2410](https://github.com/kedacore/keda/pull/2410))
- **General:** Sign KEDA images published on GitHub Container Registry ([#2501](https://github.com/kedacore/keda/pull/2501))|([#2502](https://github.com/kedacore/keda/pull/2502))|([#2504](https://github.com/kedacore/keda/pull/2504))
- **AWS Scalers:** Support temporary AWS credentials using session tokens ([#2573](https://github.com/kedacore/keda/pull/2573))
- **AWS SQS Scaler:** Allow using simple queue name instead of URL ([#2483](https://github.com/kedacore/keda/pull/2483))
- **Azure EventHub Scaler:** Don't expose connection string in metricName ([#2404](https://github.com/kedacore/keda/pull/2404))
- **Azure Pipelines Scaler:** Support `poolName` or `poolID` validation ([#2370](https://github.com/kedacore/keda/pull/2370))
- **CPU Scaler:** Adding e2e test for the cpu scaler ([#2441](https://github.com/kedacore/keda/pull/2441))
- **External Scaler:** Fix wrong calculation of retry backoff duration ([#2416](https://github.com/kedacore/keda/pull/2416))
- **Graphite Scaler:** Use the latest datapoint returned, not the earliest ([#2365](https://github.com/kedacore/keda/pull/2365))
- **Kafka Scaler:** Allow flag `topic` to be optional, where lag of all topics within the consumer group will be used for scaling ([#2409](https://github.com/kedacore/keda/pull/2409))
- **Kafka Scaler:** Concurrently query brokers for consumer and producer offsets ([#2405](https://github.com/kedacore/keda/pull/2405))
- **Kubernetes Workload Scaler:** Ignore terminated pods ([#2384](https://github.com/kedacore/keda/pull/2384))
- **PostgreSQL Scaler:** Assign PostgreSQL `userName` to correct attribute ([#2432](https://github.com/kedacore/keda/pull/2432))|([#2433](https://github.com/kedacore/keda/pull/2433))
- **Prometheus Scaler:** Support namespaced Prometheus queries ([#2575](https://github.com/kedacore/keda/issues/2575))

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
