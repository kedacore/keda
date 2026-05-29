# CHANGELOG

Inspired from [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

## [4.6.0]

### Dependencies
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.29.14 to 1.32.5 ([#707](https://github.com/opensearch-project/opensearch-go/pull/707), [#711](https://github.com/opensearch-project/opensearch-go/pull/711), [#719](https://github.com/opensearch-project/opensearch-go/pull/719), [#730](https://github.com/opensearch-project/opensearch-go/pull/730), [#737](https://github.com/opensearch-project/opensearch-go/pull/737), [#761](https://github.com/opensearch-project/opensearch-go/pull/761))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.36.4 to 1.41.0 ([#710](https://github.com/opensearch-project/opensearch-go/pull/710), [#720](https://github.com/opensearch-project/opensearch-go/pull/720), [#759](https://github.com/opensearch-project/opensearch-go/pull/759))
- Bump `github.com/stretchr/testify` from 1.10.0 to 1.11.1 ([#728](https://github.com/opensearch-project/opensearch-go/pull/728))
- Bump `github.com/aws/aws-sdk-go` from 1.55.7 to 1.55.8 ([#716](https://github.com/opensearch-project/opensearch-go/pull/716))

### Added
- Adds new fields for Opensearch 3.0 ([#702](https://github.com/opensearch-project/opensearch-go/pull/702))
- Allow users to override signing port ([#721](https://github.com/opensearch-project/opensearch-go/pull/721))
- Add `phase_took` features supported from OpenSearch 2.12 ([#722](https://github.com/opensearch-project/opensearch-go/pull/722))
- Adds the action to refresh the search analyzers to the ISM plugin ([#686](https://github.com/opensearch-project/opensearch-go/pull/686))

### Changed
- Test against Opensearch 3.0 ([#702](https://github.com/opensearch-project/opensearch-go/pull/702))
- Add more SuggestOptions to SearchResp ([#713](https://github.com/opensearch-project/opensearch-go/pull/713))
- Updates Go version to 1.24 ([#674](https://github.com/opensearch-project/opensearch-go/pull/674))
- Replace `golang.org/x/exp/slices` usage with built-in `slices` ([#674](https://github.com/opensearch-project/opensearch-go/pull/674))
- Update golangci-linter to 1.64.8 ([#740](https://github.com/opensearch-project/opensearch-go/pull/740))
- Change MaxScore to pointer ([#740](https://github.com/opensearch-project/opensearch-go/pull/740))
- Update workflow action ([#760](https://github.com/opensearch-project/opensearch-go/pull/760))
- Migrate to golangci-lint v2 ([#760](https://github.com/opensearch-project/opensearch-go/pull/760))

### Deprecated

### Removed

### Fixed
- Missing "caused by" information in StructError ([#752](https://github.com/opensearch-project/opensearch-go/pull/752))
- Add missing `ignore_unavailable`, `allow_no_indices`, and `expand_wildcards` params to MSearch ([#757](https://github.com/opensearch-project/opensearch-go/pull/757))
- Fix `UpdateResp` to correctly parse the `get` field when `_source` is requested in update operations. ([#739](https://github.com/opensearch-project/opensearch-go/pull/739))

### Security

## [4.5.0]

### Dependencies
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.29.6 to 1.29.14 ([#692](https://github.com/opensearch-project/opensearch-go/pull/692))
- Bump `github.com/aws/aws-sdk-go` from 1.55.6 to 1.55.7 ([#696](https://github.com/opensearch-project/opensearch-go/pull/696))
- Bump `github.com/wI2L/jsondiff` from 0.6.1 to 0.7.0 ([#700](https://github.com/opensearch-project/opensearch-go/pull/700))

### Added
- Adds DataStream field to IndicesGetResp struct ([#701](https://github.com/opensearch-project/opensearch-go/pull/701))
- Adds `InnerHits` field to `SearchResp` ([#672](https://github.com/opensearch-project/opensearch-go/pull/672))
- Adds `FilterPath` param ([#673](https://github.com/opensearch-project/opensearch-go/pull/673))
- Adds `Aggregations` field to `MSearchResp` ([#690](https://github.com/opensearch-project/opensearch-go/pull/690))

### Changed
- Bump golang version to 1.22 ([#691](https://github.com/opensearch-project/opensearch-go/pull/691))
- Change ChangeCatRecoveryItemResp Byte fields from int to string ([#691](https://github.com/opensearch-project/opensearch-go/pull/691))
- Changed log formatted examples code ([#694](https://github.com/opensearch-project/opensearch-go/pull/694))
- Improve the error reporting of invalid body response ([#699](https://github.com/opensearch-project/opensearch-go/pull/699))

### Deprecated

### Removed

### Fixed

### Security

## [4.4.0]

### Added
- Adds `Highlight` field to `SearchHit` ([#654](https://github.com/opensearch-project/opensearch-go/pull/654))
- Adds `MatchedQueries` field to `SearchHit` ([#663](https://github.com/opensearch-project/opensearch-go/pull/663))
- Adds support for Opensearch 2.19 ([#668](https://github.com/opensearch-project/opensearch-go/pull/668))

### Changed

### Deprecated

### Removed

### Fixed

### Security

### Dependencies
- Bump `github.com/aws/aws-sdk-go` from 1.55.5 to 1.55.6 ([#657](https://github.com/opensearch-project/opensearch-go/pull/657))
- Bump `github.com/wI2L/jsondiff` from 0.6.0 to 0.6.1 ([#643](https://github.com/opensearch-project/opensearch-go/pull/643))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.32.2 to 1.36.1 ([#664](https://github.com/opensearch-project/opensearch-go/pull/664))
- Bump `github.com/stretchr/testify` from 1.9.0 to 1.10.0 ([#644](https://github.com/opensearch-project/opensearch-go/pull/644))
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.43 to 1.29.6 ([#665](https://github.com/opensearch-project/opensearch-go/pull/665))

## [4.3.0]

### Added
- Adds ISM Alias action ([#615](https://github.com/opensearch-project/opensearch-go/pull/615))
- Adds support for opensearch 2.17 ([#623](https://github.com/opensearch-project/opensearch-go/pull/623))

### Changed

### Deprecated

### Removed

### Fixed
- Fix ISM Transition to omitempty Conditions field ([#609](https://github.com/opensearch-project/opensearch-go/pull/609))
- Fix ISM Allocation field types ([#609](https://github.com/opensearch-project/opensearch-go/pull/609))
- Fix ISM Error Notification types ([#612](https://github.com/opensearch-project/opensearch-go/pull/612))
- Fix signer receiving drained body on retries ([#620](https://github.com/opensearch-project/opensearch-go/pull/620))
- Fix Bulk Index Items not executing failure callbacks on bulk request failure ([#626](https://github.com/opensearch-project/opensearch-go/issues/626))

### Security

### Dependencies
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.31 to 1.27.43 ([#611](https://github.com/opensearch-project/opensearch-go/pull/611), [#630](https://github.com/opensearch-project/opensearch-go/pull/630), [#632](https://github.com/opensearch-project/opensearch-go/pull/632))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.32.1 to 1.32.2 ([#631](https://github.com/opensearch-project/opensearch-go/pull/631))

## [4.2.0]

### Dependencies
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.23 to 1.27.31 ([#584](https://github.com/opensearch-project/opensearch-go/pull/584), [#588](https://github.com/opensearch-project/opensearch-go/pull/588), [#593](https://github.com/opensearch-project/opensearch-go/pull/593), [#605](https://github.com/opensearch-project/opensearch-go/pull/605))
- Bump `github.com/aws/aws-sdk-go` from 1.54.12 to 1.55.5 ([#583](https://github.com/opensearch-project/opensearch-go/pull/583), [#590](https://github.com/opensearch-project/opensearch-go/pull/590), [#595](https://github.com/opensearch-project/opensearch-go/pull/595), [#596](https://github.com/opensearch-project/opensearch-go/pull/596))

### Added
- Adds `Suggest` to `SearchResp` ([#602](https://github.com/opensearch-project/opensearch-go/pull/602))
- Adds `MaxScore` to `ScrollGetResp` ([#607](https://github.com/opensearch-project/opensearch-go/pull/607))

### Changed
- Split SnapshotGetResp into sub structs  ([#603](https://github.com/opensearch-project/opensearch-go/pull/603))

### Deprecated

### Removed
- Remove workflow tests against gotip ([#604](https://github.com/opensearch-project/opensearch-go/pull/604))

### Fixed

### Security

## [4.1.0]

### Added
- Adds the `Routing` field in SearchHit interface. ([#516](https://github.com/opensearch-project/opensearch-go/pull/516))
- Adds the `SearchPipelines` field to `SearchParams` ([#532](https://github.com/opensearch-project/opensearch-go/pull/532))
- Adds support for OpenSearch 2.14 ([#552](https://github.com/opensearch-project/opensearch-go/pull/552))
- Adds the `Caches` field to Node stats ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Adds the `SeqNo` and `PrimaryTerm` fields in `SearchHit` ([#574](https://github.com/opensearch-project/opensearch-go/pull/574))
- Adds guide on configuring the client with retry and backoff ([#540](https://github.com/opensearch-project/opensearch-go/pull/540))
- Adds OpenSearch 2.15 to compatibility workflow test ([#575](https://github.com/opensearch-project/opensearch-go/pull/575))

### Changed
- Security roles get response struct has its own sub structs without omitempty ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))

### Deprecated

### Removed

### Fixed

- Fixes empty request body on retry with compression enabled ([#543](https://github.com/opensearch-project/opensearch-go/pull/543))
- Fixes `Conditions` in `PolicyStateTransition` of ISM plugin ([#556](https://github.com/opensearch-project/opensearch-go/pull/556))
- Fixes integration test response validation when response is null ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Adjust security Role struct for FLS from string to []string ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Fixes wrong response parsing for indices mapping and recovery ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Fixes wrong response parsing for security get requests ([#572](https://github.com/opensearch-project/opensearch-go/pull/572))
- Fixes opensearchtransport ignores request context cancellation when `retryBackoff` is configured ([#540](https://github.com/opensearch-project/opensearch-go/pull/540))
- Fixes opensearchtransport sleeps unexpectedly after the last retry ([#540](https://github.com/opensearch-project/opensearch-go/pull/540))
- Improves ParseError response when server response is an unknown json ([#592](https://github.com/opensearch-project/opensearch-go/pull/592))

### Security

### Dependencies
- Bump `github.com/aws/aws-sdk-go` from 1.51.21 to 1.54.12 ([#534](https://github.com/opensearch-project/opensearch-go/pull/534), [#537](https://github.com/opensearch-project/opensearch-go/pull/537), [#538](https://github.com/opensearch-project/opensearch-go/pull/538), [#545](https://github.com/opensearch-project/opensearch-go/pull/545), [#554](https://github.com/opensearch-project/opensearch-go/pull/554), [#557](https://github.com/opensearch-project/opensearch-go/pull/557), [#563](https://github.com/opensearch-project/opensearch-go/pull/563), [#564](https://github.com/opensearch-project/opensearch-go/pull/564), [#570](https://github.com/opensearch-project/opensearch-go/pull/570), [#579](https://github.com/opensearch-project/opensearch-go/pull/579))
- Bump `github.com/wI2L/jsondiff` from 0.5.1 to 0.6.0 ([#535](https://github.com/opensearch-project/opensearch-go/pull/535), [#566](https://github.com/opensearch-project/opensearch-go/pull/566))
- Bump `github.com/aws/aws-sdk-go-v2/config` from 1.27.11 to 1.27.23 ([#546](https://github.com/opensearch-project/opensearch-go/pull/546), [#553](https://github.com/opensearch-project/opensearch-go/pull/553), [#558](https://github.com/opensearch-project/opensearch-go/pull/558), [#562](https://github.com/opensearch-project/opensearch-go/pull/562), [#567](https://github.com/opensearch-project/opensearch-go/pull/567), [#571](https://github.com/opensearch-project/opensearch-go/pull/571), [#577](https://github.com/opensearch-project/opensearch-go/pull/577))
- Bump `github.com/aws/aws-sdk-go-v2` from 1.27.0 to 1.30.1 ([#559](https://github.com/opensearch-project/opensearch-go/pull/559), [#578](https://github.com/opensearch-project/opensearch-go/pull/578))

## [4.0.0]

### Added
- Adds GlobalIOUsage struct for nodes stats ([#506](https://github.com/opensearch-project/opensearch-go/pull/506))
- Adds the `Explanation` field containing the document explain details to the `SearchHit` struct. ([#504](https://github.com/opensearch-project/opensearch-go/pull/504))
- Adds new error types ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Adds handling of non json errors to ParseError ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Adds the `Failures` field to opensearchapi structs ([#510](https://github.com/opensearch-project/opensearch-go/pull/510))
- Adds the `Fields` field containing the document fields to the `SearchHit` struct. ([#508](https://github.com/opensearch-project/opensearch-go/pull/508))
- Adds security plugin ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Adds security settings to container for security testing ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Adds cluster.get-certs to copy admin certs out of the container ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Adds the `Fields` field containing stored fields to the `DocumentGetResp` struct ([#526](https://github.com/opensearch-project/opensearch-go/pull/526))
- Adds ism plugin ([#524](https://github.com/opensearch-project/opensearch-go/pull/524))

### Changed
- Uses docker compose v2 instead of v1 ([#506](https://github.com/opensearch-project/opensearch-go/pull/506))
- Updates go version to 1.21 ([#509](https://github.com/opensearch-project/opensearch-go/pull/509))
- Moves Error structs from opensearchapi to opensearch package ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Moves parseError function from opensearchapi to opensearch package as ParseError ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Changes ParseError function to do type assertion to determine error type ([#512](https://github.com/opensearch-project/opensearch-go/pull/506))
- Removes unused structs and functions from opensearch ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))
- Adjusts and extent opensearch tests for better coverage ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))
- Bumps codecov action version to v4 ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))
- Changes bulk error/reason field and some cat response fields to pointer as they can be nil ([#510](https://github.com/opensearch-project/opensearch-go/pull/510))
- Adjust workflows to work with security plugin ([#507](https://github.com/opensearch-project/opensearch-go/pull/507))
- Updates USER_GUIDE.md and /_samples/ ([#518](https://github.com/opensearch-project/opensearch-go/pull/518))
- Updates opensearchtransport.Client to use pooled gzip writer and buffer ([#521](https://github.com/opensearch-project/opensearch-go/pull/521))
- Use go:build tags for testing ([#52?](https://github.com/opensearch-project/opensearch-go/pull/52?))

### Deprecated

### Removed

### Fixed
- Fixes search request missing a slash when no indices are given ([#470](https://github.com/opensearch-project/opensearch-go/pull/469))
- Fixes opensearchtransport check for nil response body ([#517](https://github.com/opensearch-project/opensearch-go/pull/517))

### Security

### Dependencies
- Bumps `github.com/aws/aws-sdk-go-v2` from 1.25.3 to 1.26.1
- Bumps `github.com/wI2L/jsondiff` from 0.4.0 to 0.5.1
- Bumps `github.com/aws/aws-sdk-go` from 1.50.36 to 1.51.21
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.27.7 to 1.27.11

## [3.1.0]

### Added

- Adds new struct fields introduced in OpenSearch 2.12 ([#482](https://github.com/opensearch-project/opensearch-go/pull/482))
- Adds initial admin password environment variable and CI changes to support 2.12.0 release ([#449](https://github.com/opensearch-project/opensearch-go/pull/449))
- Adds `merge_id` field for indices segment request ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))

### Changed

- Updates workflow action versions ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Changes integration tests to work with secure and unsecure OpenSearch ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Moves functions from `opensearch/internal/test` to `internal/test` for more general test uses ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Changes `custom_foldername` field to pointer as it can be `null` ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Changs cat indices Primary and Replica field to pointer as it can be `null` ([#488](https://github.com/opensearch-project/opensearch-go/pull/488))
- Replaces `ioutil` with `io` in examples and integration tests [#495](https://github.com/opensearch-project/opensearch-go/pull/495)

### Fixed

- Fix incorrect SigV4 `x-amz-content-sha256` with AWS SDK v1 requests without a body ([#496](https://github.com/opensearch-project/opensearch-go/pull/496))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go` from 1.48.13 to 1.50.36
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.25.11 to 1.27.7
- Bumps `github.com/stretchr/testify` from 1.8.4 to 1.9.0

## [3.0.0]

### Added

- Adds `Err()` function to Response for detailed errors ([#246](https://github.com/opensearch-project/opensearch-go/pull/246))
- Adds golangci-lint as code analysis tool ([#313](https://github.com/opensearch-project/opensearch-go/pull/313))
- Adds govulncheck to check for go vulnerablities ([#405](https://github.com/opensearch-project/opensearch-go/pull/405))
- Adds opensearchapi with new client and function structure ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Adds integration tests for all opensearchapi functions ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Adds guide on making raw JSON REST requests ([#399](https://github.com/opensearch-project/opensearch-go/pull/399))
- Adds IPV6 support in the DiscoverNodes method ([#458](https://github.com/opensearch-project/opensearch-go/issues/458))

### Changed

- Removes the need for double error checking ([#246](https://github.com/opensearch-project/opensearch-go/pull/246))
- Updates and adjusted golangci-lint, solve linting complains for signer ([#352](https://github.com/opensearch-project/opensearch-go/pull/352))
- Solves linting complains for opensearchtransport ([#353](https://github.com/opensearch-project/opensearch-go/pull/353))
- Updates Developer guide to include docker build instructions ([#385](https://github.com/opensearch-project/opensearch-go/pull/385))
- Tests against version 2.9.0, 2.10.0, run tests in all branches, changes integration tests to wait for OpenSearch to start ([#392](https://github.com/opensearch-project/opensearch-go/pull/392))
- Makefile: uses docker golangci-lint, run integration test on `.` folder, change coverage generation ([#392](https://github.com/opensearch-project/opensearch-go/pull/392))
- golangci-lint: updates rules and fail when issues are found ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- go: updates to golang version 1.20 ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- guids: updates to work for the new opensearchapi ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Adjusts tests to new opensearchapi functions and structs ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Changes codecov to comment code coverage to each PR ([#410](https://github.com/opensearch-project/opensearch-go/pull/410))
- Changes module version from v2 to v3 ([#444](https://github.com/opensearch-project/opensearch-go/pull/444))
- Handle unexpected non-json errors with the response body ([#523](https://github.com/opensearch-project/opensearch-go/pull/523))

### Deprecated

- Deprecates legacy API `/_template` ([#390](https://github.com/opensearch-project/opensearch-go/pull/390))

### Removed

- Removes all old opensearchapi functions ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))
- Removes `/internal/build` code and folders ([#421](https://github.com/opensearch-project/opensearch-go/pull/421))

### Fixed

- Corrects AWSv4 signature on DataStream `Stats` with no index name specified ([#338](https://github.com/opensearch-project/opensearch-go/pull/338))
- Fixes GetSourceRequest `Source` field and deprecated the `Source` parameter ([#402](https://github.com/opensearch-project/opensearch-go/pull/402))
- Corrects developer guide summary with golang version 1.20 ([#434](https://github.com/opensearch-project/opensearch-go/pull/434))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go` from 1.44.263 to 1.48.13
- Bumps `github.com/aws/aws-sdk-go-v2` from 1.18.0 to 1.23.5
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.18.25 to 1.25.11
- Bumps `github.com/stretchr/testify` from 1.8.2 to 1.8.4
- Bumps `golang.org/x/net` from 0.7.0 to 0.17.0
- Bumps `github.com/golangci/golangci-lint-action` from 1.53.3 to 1.54.2

## [2.3.0]

### Added

- Adds implementation of Data Streams API ([#257](https://github.com/opensearch-project/opensearch-go/pull/257))
- Adds Point In Time API ([#253](https://github.com/opensearch-project/opensearch-go/pull/253))
- Adds InfoResp type ([#253](https://github.com/opensearch-project/opensearch-go/pull/253))
- Adds markdown linter ([#261](https://github.com/opensearch-project/opensearch-go/pull/261))
- Adds testcases to check upsert functionality ([#269](https://github.com/opensearch-project/opensearch-go/pull/269))
- Adds @Jakob3xD to co-maintainers ([#270](https://github.com/opensearch-project/opensearch-go/pull/270))
- Adds dynamic type to \_source field ([#285](https://github.com/opensearch-project/opensearch-go/pull/285))
- Adds testcases for Document API ([#285](https://github.com/opensearch-project/opensearch-go/pull/285))
- Adds `index_lifecycle` guide ([#287](https://github.com/opensearch-project/opensearch-go/pull/287))
- Adds `bulk` guide ([#292](https://github.com/opensearch-project/opensearch-go/pull/292))
- Adds `search` guide ([#291](https://github.com/opensearch-project/opensearch-go/pull/291))
- Adds `document_lifecycle` guide ([#290](https://github.com/opensearch-project/opensearch-go/pull/290))
- Adds `index_template` guide ([#289](https://github.com/opensearch-project/opensearch-go/pull/289))
- Adds `advanced_index_actions` guide ([#288](https://github.com/opensearch-project/opensearch-go/pull/288))
- Adds testcases to check UpdateByQuery functionality ([#304](https://github.com/opensearch-project/opensearch-go/pull/304))
- Adds additional timeout after cluster start ([#303](https://github.com/opensearch-project/opensearch-go/pull/303))
- Adds docker healthcheck to auto restart the container ([#315](https://github.com/opensearch-project/opensearch-go/pull/315))

### Changed

- Uses `[]string` instead of `string` in `SnapshotDeleteRequest` ([#237](https://github.com/opensearch-project/opensearch-go/pull/237))
- Updates workflows to reduce CI time, consolidate OpenSearch versions, update compatibility matrix ([#242](https://github.com/opensearch-project/opensearch-go/pull/242))
- Moves @svencowart to emeritus maintainers ([#270](https://github.com/opensearch-project/opensearch-go/pull/270))
- Reads, closes and replaces the http Response Body ([#300](https://github.com/opensearch-project/opensearch-go/pull/300))

### Fixed

- Corrects curl logging to emit the correct URL destination ([#101](https://github.com/opensearch-project/opensearch-go/pull/101))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go` from 1.44.180 to 1.44.263
- Bumps `github.com/aws/aws-sdk-go-v2` from 1.17.4 to 1.18.0
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.18.8 to 1.18.25
- Bumps `github.com/stretchr/testify` from 1.8.1 to 1.8.2

## [2.2.0]

### Added

- Adds Github workflow for changelog verification ([#172](https://github.com/opensearch-project/opensearch-go/pull/172))
- Adds Go Documentation link for the client ([#182](https://github.com/opensearch-project/opensearch-go/pull/182))
- Adds support for Amazon OpenSearch Serverless ([#216](https://github.com/opensearch-project/opensearch-go/pull/216))

### Removed

- Removes info call before performing every request ([#219](https://github.com/opensearch-project/opensearch-go/pull/219))

### Fixed

- Renames the sequence number struct tag to if_seq_no to fix optimistic concurrency control ([#166](https://github.com/opensearch-project/opensearch-go/pull/166))
- Fixes `RetryOnConflict` on bulk indexer ([#215](https://github.com/opensearch-project/opensearch-go/pull/215))

### Dependencies

- Bumps `github.com/aws/aws-sdk-go-v2` from 1.17.1 to 1.17.3
- Bumps `github.com/aws/aws-sdk-go-v2/config` from 1.17.10 to 1.18.8
- Bumps `github.com/aws/aws-sdk-go` from 1.44.176 to 1.44.180
- Bumps `github.com/aws/aws-sdk-go` from 1.44.132 to 1.44.180
- Bumps `github.com/stretchr/testify` from 1.8.0 to 1.8.1
- Bumps `github.com/aws/aws-sdk-go` from 1.44.45 to 1.44.132

[4.6.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.5.0...v4.6.0
[4.5.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.4.0...v4.5.0
[4.4.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.3.0...v4.4.0
[4.3.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.2.0...v4.3.0
[4.2.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.1.0...v4.2.0
[4.1.0]: https://github.com/opensearch-project/opensearch-go/compare/v4.0.0...v4.1.0
[4.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v3.1.0...v4.0.0
[3.1.0]: https://github.com/opensearch-project/opensearch-go/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.3.0...v3.0.0
[2.3.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/opensearch-project/opensearch-go/compare/v2.0.1...v2.1.0
[2.0.1]: https://github.com/opensearch-project/opensearch-go/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v1.1.0...v2.0.0
[1.0.0]: https://github.com/opensearch-project/opensearch-go/compare/v1.0.0...v1.1.0
