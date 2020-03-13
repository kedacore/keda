# Changelog

## v1.3

### Deprecations

- As of v1.3, support for `brokerList` is deprecated for our Kafka topic scaler and will be removed in v2.0 ([#632](https://github.com/kedacore/keda/issues/632))

### New

- Add Azure monitor scaler ([#584](https://github.com/kedacore/keda/pull/584))

### Improvements

- Make targetQueryValue configurable in postgreSQL scaler ([#643](https://github.com/kedacore/keda/pull/643))
- Added bootstrapServers to deprecate brokerList ([#621](https://github.com/kedacore/keda/pull/621))

### Breaking Changes

None.

### Other

- Changed exiting in a warning when GOROOT is not defined ([#607](https://github.com/kedacore/keda/pull/607))
- Adding Kubernetes recommended labels to resources ([#596](https://github.com/kedacore/keda/pull/596))
- Create make release target to update versions ([#610](https://github.com/kedacore/keda/pull/610))
- Update release-build.yml ([#612](https://github.com/kedacore/keda/pull/612))
- Added successful tests ([#619](https://github.com/kedacore/keda/pull/619))
- Fixed command to log keda-operator ([#622](https://github.com/kedacore/keda/pull/622))
- Update documentation to link to new Openshift 4 sample ([#625](https://github.com/kedacore/keda/pull/625))
- Change 'create' to 'push' in release action ([#627](https://github.com/kedacore/keda/pull/627))
- Document logging levels for Operator and Metrics Server ([#633](https://github.com/kedacore/keda/pull/633))
- Provide "Support" issue template ([#634](https://github.com/kedacore/keda/pull/634))
- Removed the need for deploymentName label ([#644](https://github.com/kedacore/keda/pull/644))
- Check presence of scaleTargetRef or jobTargetRef ([#648](https://github.com/kedacore/keda/pull/648))
- Add AWS pod identity support ([#499](https://github.com/kedacore/keda/pull/499))
- Updating license to Apache per CNCF donation ([#661](https://github.com/kedacore/keda/pull/661))
- Introduce changelog for KEDA ([#664](https://github.com/kedacore/keda/pull/664))
- Add vector keda logos ([#665](https://github.com/kedacore/keda/pull/665))
- readme: community call update ([#675](https://github.com/kedacore/keda/pull/675))

## v1.2

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

## v1.1

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

## v1.0

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
