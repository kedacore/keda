# Changelog

## Deprecations

- As of v1.3, support for `brokerList` is deprecated for our Kafka topic scaler and will be removed in v2.0 (#632)

## v1.2

### New

- Introduce new Postgres scaler (#553)
- Introduce new MySQL scaler (#564)

### Improvements

- TLS parameter to Redis-scaler (#540)
- Redis db index option (#577)
- Optional param for ConfigMaps and Secrets (#562)
- Remove manually adding sslmode to connection string (#558)
- ScaledObject.Status update should handle stale resource (#582)
- Improve reconcile loop (#581)
- Added SASL_SSL Plain authentication for Kafka trigger scalar to work with Event Hubs (#585)
- Address naming changes for postgresql scaler (#593)

### Breaking Changes

None.

### Other

- Move Metrics adapter into the separate Deployment (#506)
- Fix release workflow (#559)
- Improve README (#565)
- Fix gopls location (#574)
- Move release process to Markdown in repo (#569)
- Update readme steps for deploying custom KEDA (#556)
- Update the image tags for keda and keda-metrics-adapter to 1.1.0 (#549)
- Add kubernetes and platform version to the Issue template (#589)
- Add instructions on local development and debugging (#583)
- Proposal for PR template (#586)
- Add a checkenv target (#600)
- Added links to the scaler interface documentation (#597)
- Correcting release process doc (#602)
- Mentioning problem with checksum mismatch error (#605)
- Local deployment minor fix (#603)