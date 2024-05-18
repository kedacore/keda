# v1.31.1 (2024-03-29)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.31.0 (2024-03-20)

* **Feature**: This release introduces 3 new APIs ('GetResourcePolicy', 'PutResourcePolicy' and 'DeleteResourcePolicy') and modifies the existing 'CreateTable' API for the resource-based policy support. It also modifies several APIs to accept a 'TableArn' for the 'TableName' parameter.

# v1.30.5 (2024-03-18)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.30.4 (2024-03-07)

* **Bug Fix**: Remove dependency on go-cmp.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.30.3 (2024-03-06)

* **Documentation**: Doc only updates for DynamoDB documentation

# v1.30.2 (2024-03-04)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.30.1 (2024-02-23)

* **Bug Fix**: Move all common, SDK-side middleware stack ops into the service client module to prevent cross-module compatibility issues in the future.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.30.0 (2024-02-22)

* **Feature**: Add middleware stack snapshot tests.

# v1.29.2 (2024-02-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.29.1 (2024-02-20)

* **Bug Fix**: When sourcing values for a service's `EndpointParameters`, the lack of a configured region (i.e. `options.Region == ""`) will now translate to a `nil` value for `EndpointParameters.Region` instead of a pointer to the empty string `""`. This will result in a much more explicit error when calling an operation instead of an obscure hostname lookup failure.
* **Documentation**: Publishing quick fix for doc only update.

# v1.29.0 (2024-02-16)

* **Feature**: Add new ClientOptions field to waiter config which allows you to extend the config for operation calls made by waiters.

# v1.28.1 (2024-02-15)

* **Bug Fix**: Correct failure to determine the error type in awsJson services that could occur when errors were modeled with a non-string `code` field.

# v1.28.0 (2024-02-13)

* **Feature**: Bump minimum Go version to 1.20 per our language support policy.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.1 (2024-02-02)

* **Documentation**: Any number of users can execute up to 50 concurrent restores (any type of restore) in a given account.

# v1.27.0 (2024-01-19)

* **Feature**: This release adds support for including ApproximateCreationDateTimePrecision configurations in EnableKinesisStreamingDestination API, adds the same as an optional field in the response of DescribeKinesisStreamingDestination, and adds support for a new UpdateKinesisStreamingDestination API.

# v1.26.9 (2024-01-17)

* **Documentation**: Updating note for enabling streams for UpdateTable.

# v1.26.8 (2024-01-04)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.26.7 (2023-12-20)

* No change notes available for this release.

# v1.26.6 (2023-12-08)

* **Bug Fix**: Reinstate presence of default Retryer in functional options, but still respect max attempts set therein.

# v1.26.5 (2023-12-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.26.4 (2023-12-06)

* **Bug Fix**: Restore pre-refactor auth behavior where all operations could technically be performed anonymously.

# v1.26.3 (2023-12-01)

* **Bug Fix**: Correct wrapping of errors in authentication workflow.
* **Bug Fix**: Correctly recognize cache-wrapped instances of AnonymousCredentials at client construction.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.26.2 (2023-11-30.2)

* **Bug Fix**: Respect caller region overrides in endpoint discovery.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.26.1 (2023-11-30)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.26.0 (2023-11-29)

* **Feature**: Expose Options() accessor on service clients.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.25.5 (2023-11-28.2)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.25.4 (2023-11-28)

* **Bug Fix**: Respect setting RetryMaxAttempts in functional options at client construction.

# v1.25.3 (2023-11-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.25.2 (2023-11-15)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.25.1 (2023-11-09)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.25.0 (2023-11-01)

* **Feature**: Adds support for configured endpoints via environment variables and the AWS shared configuration file.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.24.0 (2023-10-31)

* **Feature**: **BREAKING CHANGE**: Bump minimum go version to 1.19 per the revised [go version support policy](https://aws.amazon.com/blogs/developer/aws-sdk-for-go-aligns-with-go-release-policy-on-supported-runtimes/).
* **Dependency Update**: Updated to the latest SDK module versions

# v1.23.0 (2023-10-18)

* **Feature**: Add handwritten paginators that were present in some services in the v1 SDK.
* **Documentation**: Updating descriptions for several APIs.

# v1.22.2 (2023-10-12)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.22.1 (2023-10-06)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.22.0 (2023-09-26)

* **Feature**: Amazon DynamoDB now supports Incremental Export as an enhancement to the existing Export Table

# v1.21.5 (2023-08-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.21.4 (2023-08-18)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.21.3 (2023-08-17)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.21.2 (2023-08-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.21.1 (2023-08-01)

* No change notes available for this release.

# v1.21.0 (2023-07-31)

* **Feature**: Adds support for smithy-modeled endpoint resolution. A new rules-based endpoint resolution will be added to the SDK which will supercede and deprecate existing endpoint resolution. Specifically, EndpointResolver will be deprecated while BaseEndpoint and EndpointResolverV2 will take its place. For more information, please see the Endpoints section in our Developer Guide.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.20.3 (2023-07-28)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.20.2 (2023-07-25)

* **Documentation**: Documentation updates for DynamoDB

# v1.20.1 (2023-07-13)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.20.0 (2023-06-29)

* **Feature**: This release adds ReturnValuesOnConditionCheckFailure parameter to PutItem, UpdateItem, DeleteItem, ExecuteStatement, BatchExecuteStatement and ExecuteTransaction APIs. When set to ALL_OLD,  API returns a copy of the item as it was when a conditional write failed

# v1.19.11 (2023-06-21)

* **Documentation**: Documentation updates for DynamoDB

# v1.19.10 (2023-06-15)

* No change notes available for this release.

# v1.19.9 (2023-06-13)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.19.8 (2023-06-12)

* **Documentation**: Documentation updates for DynamoDB

# v1.19.7 (2023-05-04)

* No change notes available for this release.

# v1.19.6 (2023-04-24)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.19.5 (2023-04-17)

* **Documentation**: Documentation updates for DynamoDB API

# v1.19.4 (2023-04-10)

* No change notes available for this release.

# v1.19.3 (2023-04-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.19.2 (2023-03-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.19.1 (2023-03-10)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.19.0 (2023-03-08)

* **Feature**: Adds deletion protection support to DynamoDB tables. Tables with deletion protection enabled cannot be deleted. Deletion protection is disabled by default, can be enabled via the CreateTable or UpdateTable APIs, and is visible in TableDescription. This setting is not replicated for Global Tables.

# v1.18.6 (2023-03-03)

* **Documentation**: Documentation updates for DynamoDB.

# v1.18.5 (2023-02-22)

* **Bug Fix**: Prevent nil pointer dereference when retrieving error codes.

# v1.18.4 (2023-02-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.18.3 (2023-02-15)

* **Announcement**: When receiving an error response in restJson-based services, an incorrect error type may have been returned based on the content of the response. This has been fixed via PR #2012 tracked in issue #1910.
* **Bug Fix**: Correct error type parsing for restJson services.

# v1.18.2 (2023-02-03)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.18.1 (2023-01-23)

* No change notes available for this release.

# v1.18.0 (2023-01-05)

* **Feature**: Add `ErrorCodeOverride` field to all error structs (aws/smithy-go#401).

# v1.17.9 (2022-12-15)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.17.8 (2022-12-02)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.17.7 (2022-11-22)

* No change notes available for this release.

# v1.17.6 (2022-11-18)

* **Documentation**: Updated minor fixes for DynamoDB documentation.

# v1.17.5 (2022-11-16)

* No change notes available for this release.

# v1.17.4 (2022-11-10)

* No change notes available for this release.

# v1.17.3 (2022-10-24)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.17.2 (2022-10-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.17.1 (2022-09-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.17.0 (2022-09-15)

* **Feature**: Increased DynamoDB transaction limit from 25 to 100.

# v1.16.5 (2022-09-14)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.16.4 (2022-09-02)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.16.3 (2022-08-31)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.16.2 (2022-08-30)

* No change notes available for this release.

# v1.16.1 (2022-08-29)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.16.0 (2022-08-18)

* **Feature**: This release adds support for importing data from S3 into a new DynamoDB table

# v1.15.13 (2022-08-11)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.12 (2022-08-09)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.11 (2022-08-08)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.10 (2022-08-01)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.9 (2022-07-05)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.8 (2022-06-29)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.7 (2022-06-17)

* **Documentation**: Doc only update for DynamoDB service

# v1.15.6 (2022-06-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.5 (2022-05-17)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.4 (2022-04-25)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.3 (2022-03-30)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.2 (2022-03-24)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.1 (2022-03-23)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.0 (2022-03-08)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.14.0 (2022-02-24)

* **Feature**: API client updated
* **Feature**: Adds RetryMaxAttempts and RetryMod to API client Options. This allows the API clients' default Retryer to be configured from the shared configuration files or environment variables. Adding a new Retry mode of `Adaptive`. `Adaptive` retry mode is an experimental mode, adding client rate limiting when throttles reponses are received from an API. See [retry.AdaptiveMode](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/retry#AdaptiveMode) for more details, and configuration options.
* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.13.0 (2022-01-14)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.12.0 (2022-01-07)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.11.0 (2021-12-21)

* **Feature**: API Paginators now support specifying the initial starting token, and support stopping on empty string tokens.
* **Feature**: Updated to latest service endpoints

# v1.10.0 (2021-12-02)

* **Feature**: API client updated
* **Bug Fix**: Fixes a bug that prevented aws.EndpointResolverWithOptions from being used by the service client. ([#1514](https://github.com/aws/aws-sdk-go-v2/pull/1514))
* **Dependency Update**: Updated to the latest SDK module versions

# v1.9.0 (2021-11-30)

* **Feature**: API client updated
* **Dependency Update**: Updated to the latest SDK module versions

# v1.8.1 (2021-11-19)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.8.0 (2021-11-12)

* **Feature**: Service clients now support custom endpoints that have an initial URI path defined.
* **Feature**: Waiters now have a `WaitForOutput` method, which can be used to retrieve the output of the successful wait operation. Thank you to [Andrew Haines](https://github.com/haines) for contributing this feature.
* **Documentation**: Updated service to latest API model.

# v1.7.0 (2021-11-06)

* **Feature**: The SDK now supports configuration of FIPS and DualStack endpoints using environment variables, shared configuration, or programmatically.
* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.6.0 (2021-10-21)

* **Feature**: API client updated
* **Feature**: Updated  to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.5.2 (2021-10-11)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.5.1 (2021-09-17)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.5.0 (2021-08-27)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.4.3 (2021-08-19)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.4.2 (2021-08-04)

* **Dependency Update**: Updated `github.com/aws/smithy-go` to latest version.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.4.1 (2021-07-15)

* **Dependency Update**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.4.0 (2021-06-25)

* **Feature**: Adds support for endpoint discovery.
* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.3.1 (2021-05-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.3.0 (2021-05-14)

* **Feature**: Constant has been added to modules to enable runtime version inspection for reporting.
* **Dependency Update**: Updated to the latest SDK module versions

