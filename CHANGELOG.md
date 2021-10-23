# Changelog

## Deprecations

## History

- [Unreleased](#unreleased)
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

- TODO ([#XXX](https://github.com/kedacore/keda/pull/XXX))

### Improvements

- TODO ([#XXX](https://github.com/kedacore/keda/pull/XXX))

### Breaking Changes

- TODO ([#XXX](https://github.com/kedacore/keda/pull/XXX))

### Other

- TODO ([#XXX](https://github.com/kedacore/keda/pull/XXX))

## v2.3.0

### New

- Add Cassandra Scaler ([#1817](https://github.com/kedacore/keda/pull/1817))
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
