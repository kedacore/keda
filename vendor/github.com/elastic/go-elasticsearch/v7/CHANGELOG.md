# v7.17.1

## Client

* Fixed serialisation of the `routing` parameter for the `BulkIndexerItem` routing property.

# v7.17.0

## Client

* Fixed a race condition on metrics in transport [#397](https://github.com/elastic/go-elasticsearch/pull/397), thanks @mainliufeng !
* The client will now return an error if a required arguments is passed as a nil value. [#201](https://github.com/elastic/go-elasticsearch/issues/201)
* API is compatible with Elasticsearch 7.17.0

# v8.0.0-alpha

## Client

### Elastic Transport
* This is the first release using the all new `elastictransport` which now lives in the [elastic-transport-go](https://github.com/elastic/elastic-transport-go/) repository. The goal is to allow for reuse between this and future Go clients.
### API
* Changed the `Body` parameter for `BulkIndexerItem` in favor of `io.ReadSeeker` to lower memory consumption when using the `esutil.BulkIndexer`.
* Replaced the `Config` option `RetryOnTimeout` in favor of a new `RetryOnError` function which allows to dynamically chose what error should be retried."

# v7.16.0

# Client
* Adds versioning and routing options to the `BulkIndexer`. Thanks to @mehran-prs and @munkyboy !

* Adds CA fingerprinting.  You can configure the client to only trust certificates that are signed by a specific CA certificate (CA certificate pinning) by providing a ca_fingerprint option. This will verify that the fingerprint of the CA certificate that has signed the certificate of the server matches the supplied value:
```go
elasticsearch.NewClient(Config{
Addresses:              []string{\"https://elastic:changeme@localhost:9200\"},
CertificateFingerprint: \"A6FB224A4386...\"
})
```
# API

* New APIs:
  * ModifyDataStream, [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/master/data-streams.html)
  * TransformUpgradeTransforms, [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/upgrade-transforms.html)
  * Migration.GetFeatureUpgradeStatus, [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/migration-api-feature-upgrade.html)
  * Migration.PostFeatureUpgrade, [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/migration-api-feature-upgrade.html)


* New **Experimental** API:

  * FleetSearch, [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/7.16/fleet-search.html)
  * FleetMsearch, [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/7.16/fleet-multi-search.html)


* Changes:
  * DeleteByQuery 
    * Removed _source, _source_excludes, _source_includes parameters.
  * UpdateByQuery 
    * Removed _source, _source_excludes, _source_includes parameters.
  * OpenPointInTime
    * The parameter keep_alive is now required.
  * SearchMvt
    * Added TrackTotalHits, Indicate if the number of documents that match the query should be tracked. A number can also be specified, to accurately track the total hit count up to the number.
  * IngestPutPipeline
    * Added WithIfVersion, required version for optimistic concurrency control for pipeline updates.
  * IndicesGetIndexTemplate
    * WithName, a pattern that returned template names must match.
  * NodesHotThreads
    * WithSort, the sort order for 'cpu' type (default: total).
  * MLPutTrainedModel
    * WithDeferDefinitionDecompression, if set to `true` and a `compressed_definition` is provided, the request defers definition decompression and skips relevant validations.
  * TransformDeleteTransform
    * WithTimeout, controls the time to wait for the transform deletion.
  * TransformPutTransform
    * WithTimeout, controls the time to wait for the transform to start.
  * TransformUpdateTransform
    * WithTimeout, controls the time to wait for the update.

* Promoted to stable:
  * FleetGlobalCheckpoints
  * GetScriptContext
  * GetScriptLanguages
  * IndicesResolveIndex
  * MonitoringBulk
  * RankEval
  * SearchableSnapshotsMount
  * SearchableSnapshotsStats
  * SecurityClearCachedServiceTokens
  * SecurityCreateServiceToken
  * SecurityDeleteServiceToken
  * SecurityGetServiceAccounts
  * SecurityGetServiceCredentials
  * ShutdownDeleteNode
  * ShutdownGetNode
  * ShutdownPutNode
  * TermsEnum"

# v7.15.1

# Client
  * Allow User-Agent override via the `userAgentHeader` header. Credit goes to @aleksmaus!  "

# v7.15.0

# Client
  * Body compression can now be enabled in the client via the `CompressRequestBody` config option. Thank you @bschofield for this contribution ! 
  * # API
 
  * New APIs:
    * Security
    * QueryAPIKeys [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/7.15/security-api-query-api-key.html)
 
  * New **Experimental** API:
    * Indices
      * DiskUsage [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/master/indices-disk-usage.html)
      * FieldUsageStats [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/master/field-usage-stats.html)
    * Nodes
      * ClearRepositoriesMeteringArchive [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/7.15/clear-repositories-metering-archive-api.html)
      * GetRepositoriesMeteringInfo [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/7.15/get-repositories-metering-api.html)
    * SearchMvt [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/master/search-vector-tile-api.html)"

# v7.14.0

# Client
Starting in v7.14.0 the client performs a required product check before the first API call is executed. This product check allows the client to establish that it’s communicating with a supported Elasticsearch cluster.

The product check requires a single HTTP request to the `info` API. In most cases this request will succeed quickly and then no further product check HTTP requests will be sent.

# API
* New APIs:
  * ILM
    * MigrateToDataTiers
  * ML
    * ResetJob
  * SAML
    * SamlAuthenticate           
    * SamlCompleteLogout         
    * SamlInvalidate             
    * SamlLogout                 
    * SamlPrepareAuthentication  
    * SamlServiceProviderMetadata
  * SQL
    * DeleteAsync   
    * GetAsync      
    * GetAsyncStatus

* New **Beta** API:
  * TermsEnum, [see documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/search-terms-enum.html)


# v7.13.1

# X-Pack
* New API:
  * `SnapshotRepositoryAnalyze`"

# v7.13.0

# Client
* Adds support for compatibility header for Elasticsearch. If the environment variable 'ELASTIC_CLIENT_APIVERSIONING' is set to true or 1, the client will send the headers Accept and Content-Type with the following value: application/vnd.elasticsearch+json;compatible-with=7.
* Favor `POST` method when only `GET` & `POST` method are available to prevent goroutine leak. https://github.com/elastic/go-elasticsearch/issues/250
* Filter master only nodes in discovery. https://github.com/elastic/go-elasticsearch/issues/256

# API

* New APIs: 
  * `FeaturesResetFeatures`
  * `IngestGeoIPStats`

* New experimental APIs: 
  * `ShutdownGetNode`
  * `ShutdownPutNode`
  * `ShutdownDeleteNode`

# X-Pack

* New APIs:
  * `MLDeleteTrainedModelAlias`
  * `MLPreviewDataFrameAnalytics`
  * `MLPutTrainedModelAlias`

* APIs promoted to stable:
  * `TextStructureFindStructure`
  * `MLDeleteDataFrameAnalytics`
  * `MLDeleteTrainedModel`
  * `MLEvaluateDataFrame`
  * `MLExplainDataFrameAnalytics`
  * `MLGetDataFrameAnalytics`
  * `MLGetDataFrameAnalyticsStats`
  * `MLGetTrainedModels`
  * `MLGetTrainedModelsStats`
  * `MLPutDataFrameAnalytics`
  * `MLPutTrainedModel`
  * `MLStartDataFrameAnalytics`
  * `MLStopDataFrameAnalytics`
  * `MLUpdateDataFrameAnalytics`

* New Beta APIs:
  * `SecurityCreateServiceToken`
  * `SecurityClearCachedServiceTokens`
  * `SecurityDeleteServiceToken`
  * `SecurityGetServiceAccounts`
  * `SecurityGetServiceCredentials`

* New experimental APIs: 
  * `SearchableSnapshotsCacheStats`

# v7.12.0

# Transport
* Added the `X-Elastic-Client-Meta` HTTP header (#240)
* Fixed of by one error in the retry mechanism of the client (#242)

# API
## New
* `GetFeatures` within `Snapshot.GetFeatures` & `FeaturesGetFeatures` 

## Added
* `Search` with `MinCompatibleShardNode` - the minimum compatible version that all shards involved in search should have for this request to be successful.

# X-Pack
## New
* `EqlGetStatus` - Returns the status of a previously submitted async or stored Event Query Language (EQL) search
* `Logstash` with `LogstashGetPipeline` `LogstashPutPipeline` and `LogstashDeletePipeline` [More info](https://www.elastic.co/guide/en/elasticsearch/reference/current/logstash-apis.html)
* `TextStructureFindStructure` - Finds the structure of a text file. The text file must contain data that is suitable to be ingested into Elasticsearch.
* `AutoscalingGetAutoscalingPolicy`, `AutoscalingPutAutoscalingPolicy`, `AutoscalingDeleteAutoscalingPolicy` and `AutoscalingGetAutoscalingCapacity` are promoted `stable` 

## Added
* `SearchableSnapshotsStats` with `WithLevel` - return stats aggregated at cluster, index or shard level.
* `SearchableSnapshotsMount` with `WithStorage` - selects the kind of local storage used to accelerate searches. experimental, and defaults to `full_copy`.


# v7.11.0

* Updated APIs for Elasticsearch 7.11"

# v7.10.0

* Updated APIs for Elasticsearch 7.10 
* Fixed capitalization of UUID values"

# v7.9.0

* Updated APIs for Elasticsearch 7.9
* Util: Reinstate item.Body after it is consumed in BulkIndexer
* Transport: Fix memory leak when retrying 5xx responses
* Fixes and improvements to the test generator

# v7.8.0

* Updated APIs for Elasticsearch 7.8.0

# v7.7.0

* API: Add convenience method for accessing the deprecation warnings in the response headers
* Transport: Add the CACert global configuration option
* Transport: Add support for global request headers
* Util: Add the BulkIndexer helper; see [example](https://github.com/elastic/go-elasticsearch/tree/master/_examples/bulk#indexergo)
* Examples: Add examples and benchmarks for the BulkIndexer helper
* CI: Add workflows for Github Actions
* CI: Remove Travis CI
* Generator: Tests: Fixes and improvements
* Generator: Source: Updates and improvements"

# v7.6.0

* Ignore the ELASTICSEARCH_URL variable when address is passed in configuration
* Retry on EOF errors"

# v6.8.5

* Support for Elasticsearch 6.8.5 APIs
* Add support for request retries
* Add connection state management
* Add support for node discovery in client

# v7.5.0

* Support for Elasticsearch 7.5 APIs
* Add support for request retries
* Add connection state management
* Add support for node discovery in client

