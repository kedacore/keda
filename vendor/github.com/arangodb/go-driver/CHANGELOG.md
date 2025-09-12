# Change Log

## [master](https://github.com/arangodb/go-driver/tree/master) (N/A)

## [1.6.6](https://github.com/arangodb/go-driver/tree/v1.6.6) (2025-02-21)
- Switch to Go 1.22.11
- Switch to jwt-go v5

## [1.6.5](https://github.com/arangodb/go-driver/tree/v1.6.5) (2024-11-15)
- Expose `NewType` method
- Switch to Go 1.22.8

## [1.6.4](https://github.com/arangodb/go-driver/tree/v1.6.4) (2024-09-27)
- Switch to Go 1.22.5
- Switch to Go 1.22.6

## [1.6.2](https://github.com/arangodb/go-driver/tree/v1.6.2) (2024-04-02)
- Switch to Go 1.20.11
- Switch to Go 1.21.5
- Disable AF mode in tests (not supported since 3.12)
- Remove graph with all collections
- Allow skipping validation for Database and Collection existence
- Deprecate Pregel Job API
- `MDI` and `MDI-Prefixed` indexes. Deprecate `ZKD` index

## [1.6.1](https://github.com/arangodb/go-driver/tree/v1.6.1) (2023-10-31)
- Add support for getting license
- Add support for Raw Authentication in VST (support external jwt token as raw element)
- Fix race when using WithRawResponse/WithResponse context with agencyConnection 
- Async Client
- Expose getters for Context values
- Deprecate `AllowInconsistent` in HotBackup
- Revert ReturnOld for edge/vertex operations
- Agency: Deprecate TTL and observe features
- Bugfix: Force analyzer removal
- Move examples to separate package
- Deprecate ClientConfig.SynchronizeEndpointsInterval due to bug in implementation
- Add Rename function for collections (single server only).
- Fix using VST for database with non-ANSI characters
- Automate release process

## [1.6.0](https://github.com/arangodb/go-driver/tree/v1.6.0) (2023-05-30)
- Add ErrArangoDatabaseNotFound and IsExternalStorageError helper to v2
- [V2] Support for Collection Documents removal
- [V2] Fix: Plain Connection doesn't work with JWT authentication
- Support for new error codes if write concern is not fulfilled
- Support for geo_s2 analyzers
- Add replication V2 option for database creation
- Use Go 1.20.3 for testing. Add govulncheck to pipeline
- Fix test for extended names
- Fix potential bug with DB name escaping for URL when requesting replication-related API
- Retriable batch reads in AQL cursors
- Add support for explain API ([v1] and [V2])
- Search optimisation for inverted index and ArangoSearch
- [V2] Fix AF mode in tests
- Support for optimizer rules in AQL query
- Add support for refilling index caches
- [V2] Retriable batch reads in AQL cursors
- Add log level support for a specific server
- Allow for VPACK encoding in _api/gharial API

## [1.5.2](https://github.com/arangodb/go-driver/tree/v1.5.2) (2023-03-01)
- Bump `DRIVER_VERSION`

## [1.5.1](https://github.com/arangodb/go-driver/tree/v1.5.1) (2023-03-01)
- Add `x-arango-driver` header flag

## [1.5.0](https://github.com/arangodb/go-driver/tree/v1.5.0) (2023-02-17)
- Use Go 1.19.4
- Add `IsExternalStorageError` to check for [external storage errors](https://docs.arangodb.com/stable/develop/error-codes-and-meanings/#external-arangodb-storage-errors)
- `nested` field in arangosearch type View
- Fix: TTL index creation fails when expireAt is 0
- [V2] Support for Collection Indexes
- Fix: Fetching single InvertedIndex fails with Marshalling error

## [1.4.1](https://github.com/arangodb/go-driver/tree/v1.4.1) (2022-12-14)
- Add support for `checksum` in Collections
- Fix reusing same connection with different Authentication parameters passed via driver.NewClient
- Add `cache` for ArangoSearchView Link and StoredValue types and `primarySortCache`, `primaryKeyCache` for ArangoSearchView type

## [1.4.0](https://github.com/arangodb/go-driver/tree/v1.4.0) (2022-10-04)
- Add `hex` property to analyzer's properties
- Add support for `computedValues`
- Optional `computeOn` field in `computedValues`
- Add support for `computedValues` into collection inventory
- Update the structures to align them with the ArangoDB 3.10 release
- Add `IsNotFoundGeneral` and `IsDataSourceOrDocumentNotFound` methods - deprecate `IsNotFound`
- Add support for optimizer rules (AQL query)
- New `LegacyPolygons` parameter for Geo Indexes
- New parameters (`cacheEnabled` and `storedValues`) for Persistent Indexes
- New analyzers: `classification`, `nearest neighbors`, `minhash`
- Add support for Inverted index
- Deprecate fulltext index
- Add support for Pregel API
- Add tests to check support for Enterprise Graphs
- Search View v2 (`search-alias`)
- Add Rename View support
- Add support for `Metrics`

## [1.3.3](https://github.com/arangodb/go-driver/tree/v1.3.3) (2022-07-27)
- Fix `lastValue` field type
- Setup Go-lang linter with minimal configuration
- Use Go 1.17.6
- Add missing `deduplicate` param to PersistentIndex

## [1.3.2](https://github.com/arangodb/go-driver/tree/v1.3.2) (2022-05-16)
- Fix selectivityEstimate Index field type

## [1.3.1](https://github.com/arangodb/go-driver/tree/v1.3.1) (2022-03-23)
- Add support for `exclusive` field for transaction options
- Fix cursor executionTime statistics getter
- Fix cursor warnings field type
- Fix for DocumentMeta name field overrides name field

## [1.3.0](https://github.com/arangodb/go-driver/tree/v1.3.0) (2022-03-17)
- Disallow unknown fields feature
- inBackground parameter in ArangoSearch links
- ZKD indexes
- Hybrid SmartGraphs
- Segmentation and Collation Analyzers
- Bypass caching for specific collections
- Overload Control
- [V2] Add support for streaming the response body by the caller.
- [V2] Bugfix with escaping the URL path twice.
- Bugfix for the satellites' collection shard info.
- [V2] Support for satellites' collections.

## [1.2.1](https://github.com/arangodb/go-driver/tree/v1.2.1) (2021-09-21)
- Add support for fetching shards' info by the given collection name.
- Change versioning to be go mod compatible
- Add support for ForceOneShardAttributeValue in Query

## [1.2.0](https://github.com/arangodb/go-driver/tree/1.2.0) (2021-08-04)
- Add support for AQL, Pipeline, Stopwords, GeoJSON and GeoPoint Arango Search analyzers.
- Add `estimates` field to indexes properties.
- Add tests for 3.8 ArangoDB and remove tests for 3.5.
- Add Plan support in Query execution.
- Change Golang version from 1.13.4 to 1.16.6.
- Add graceful shutdown for the coordinators.
- Replace 'github.com/dgrijalva/jwt-go' with 'github.com/golang-jwt/jwt'

## [1.1.1](https://github.com/arangodb/go-driver/tree/1.1.1) (2020-11-13)
- Add Driver V2 in Alpha version
- Add HTTP2 support for V1 and V2
- Don't omit the `stopwords` field. The field is mandatory in 3.6 ArangoDB

## [1.1.0](https://github.com/arangodb/go-driver/tree/1.1.0) (2020-08-11)
- Use internal coordinator communication for cursors if specified coordinator was not found on endpoint list
- Add support for Overwrite Mode (ArangoDB 3.7)
- Add support for Schema Collection options (ArangoDB 3.7)
- Add support for Disjoint and Satellite Graphs options (ArangoDB 3.7)

## [1.0.0](https://github.com/arangodb/go-driver/tree/1.0.0) (N/A)
- Enable proper CHANGELOG and versioning
