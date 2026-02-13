# github.com/solarwinds/swo-sdk-go/v1

Developer-friendly & type-safe Go SDK specifically catered to leverage *github.com/solarwinds/swo-sdk-go/v1* API.

<div align="left">
    <a href="https://www.speakeasy.com/?utm_source=github-com/solarwinds/swo-sdk-go/v1&utm_campaign=go"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://opensource.org/licenses/MIT">
        <img src="https://img.shields.io/badge/License-MIT-blue.svg" style="width: 100px; height: 28px;" />
    </a>
</div>


<br /><br />
> [!IMPORTANT]
> This SDK is not yet ready for production use. To complete setup please follow the steps outlined in your [workspace](https://app.speakeasy.com/org/swo/swo). Delete this section before > publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary

SolarWinds Observability: SolarWinds Observability REST API
[Rest API Documentation](https://documentation.solarwinds.com/en/success_center/observability/content/api/api-swagger.htm)
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [github.com/solarwinds/swo-sdk-go/v1](#githubcomsolarwindsswo-sdk-gov1)
  * [SDK Installation](#sdk-installation)
  * [SDK Example Usage](#sdk-example-usage)
  * [Authentication](#authentication)
  * [Available Resources and Operations](#available-resources-and-operations)
  * [Pagination](#pagination)
  * [Retries](#retries)
  * [Error Handling](#error-handling)
  * [Server Selection](#server-selection)
  * [Custom HTTP Client](#custom-http-client)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start SDK Installation [installation] -->
## SDK Installation

To add the SDK as a dependency to your project:
```bash
go get github.com/solarwinds/swo-sdk-go/swov1
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->

<!-- Start Authentication [security] -->
## Authentication

### Per-Client Security Schemes

This SDK supports the following security scheme globally:

| Name       | Type | Scheme      | Environment Variable |
| ---------- | ---- | ----------- | -------------------- |
| `APIToken` | http | HTTP Bearer | `SWO_API_TOKEN`      |

You can configure it using the `WithSecurity` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```
<!-- End Authentication [security] -->

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

<details open>
<summary>Available methods</summary>

### [ChangeEvents](docs/sdks/changeevents/README.md)

* [CreateChangeEvent](docs/sdks/changeevents/README.md#createchangeevent) - Create an event

### [CloudAccounts](docs/sdks/cloudaccounts/README.md)

* [ActivateAwsIntegration](docs/sdks/cloudaccounts/README.md#activateawsintegration) - Activate AWS Integration
* [CreateOrgStructure](docs/sdks/cloudaccounts/README.md#createorgstructure) - Create Organizational Structure
* [UpdateAwsIntegration](docs/sdks/cloudaccounts/README.md#updateawsintegration) - Update AWS Integration
* [ValidateMgmtAccountOnboarding](docs/sdks/cloudaccounts/README.md#validatemgmtaccountonboarding) - Validate Management Account Onboarding

### [Dbo](docs/sdks/dbo/README.md)

* [ObserveDatabase](docs/sdks/dbo/README.md#observedatabase) - Add database observability to a database
* [GetConfig](docs/sdks/dbo/README.md#getconfig) - Get organization-level configuration for database observability agents/plugins
* [SetConfig](docs/sdks/dbo/README.md#setconfig) - Set organization-level configuration for database observability agents/plugins
* [GetPublicKey](docs/sdks/dbo/README.md#getpublickey) - Get public key for encrypting database credentials locally
* [UpdateDatabase](docs/sdks/dbo/README.md#updatedatabase) - Update an observed database
* [DeleteDatabase](docs/sdks/dbo/README.md#deletedatabase) - Delete an observed database
* [GetPluginConfig](docs/sdks/dbo/README.md#getpluginconfig) - Get configuration of plugins observing a database
* [GetPlugins](docs/sdks/dbo/README.md#getplugins) - Get status of plugins observing a database
* [PluginOperation](docs/sdks/dbo/README.md#pluginoperation) - Apply an operation on a database observability plugin

### [Dem](docs/sdks/dem/README.md)

* [ListProbes](docs/sdks/dem/README.md#listprobes) - Get a list of existing synthetic probes
* [GetDemSettings](docs/sdks/dem/README.md#getdemsettings) - Get DEM settings
* [SetDemSettings](docs/sdks/dem/README.md#setdemsettings) - Set DEM settings
* [CreateTransaction](docs/sdks/dem/README.md#createtransaction) - Create transaction monitoring configuration
* [GetTransaction](docs/sdks/dem/README.md#gettransaction) - Get transaction monitoring configuration
* [UpdateTransaction](docs/sdks/dem/README.md#updatetransaction) - Update transaction monitoring configuration
* [DeleteTransaction](docs/sdks/dem/README.md#deletetransaction) - Delete transaction
* [PauseTransactionMonitoring](docs/sdks/dem/README.md#pausetransactionmonitoring) - Pause monitoring of the transaction
* [UnpauseTransactionMonitoring](docs/sdks/dem/README.md#unpausetransactionmonitoring) - Unpause monitoring of the transaction
* [CreateURI](docs/sdks/dem/README.md#createuri) - Create URI monitoring configuration
* [GetURI](docs/sdks/dem/README.md#geturi) - Get URI monitoring configuration
* [UpdateURI](docs/sdks/dem/README.md#updateuri) - Update URI monitoring configuration
* [DeleteURI](docs/sdks/dem/README.md#deleteuri) - Delete URI
* [PauseURIMonitoring](docs/sdks/dem/README.md#pauseurimonitoring) - Pause monitoring of the URI
* [UnpauseURIMonitoring](docs/sdks/dem/README.md#unpauseurimonitoring) - Unpause monitoring of the URI
* [CreateWebsite](docs/sdks/dem/README.md#createwebsite) - Create website monitoring configuration
* [GetWebsite](docs/sdks/dem/README.md#getwebsite) - Get website monitoring configuration
* [UpdateWebsite](docs/sdks/dem/README.md#updatewebsite) - Update website monitoring configuration
* [DeleteWebsite](docs/sdks/dem/README.md#deletewebsite) - Delete website
* [PauseWebsiteMonitoring](docs/sdks/dem/README.md#pausewebsitemonitoring) - Pause monitoring of a website
* [UnpauseWebsiteMonitoring](docs/sdks/dem/README.md#unpausewebsitemonitoring) - Unpause monitoring of a website

### [Entities](docs/sdks/entities/README.md)

* [ListEntities](docs/sdks/entities/README.md#listentities) - Get a list of entities by type. A returned empty list indicates no entities matched the given parameters.
* [GetEntityByID](docs/sdks/entities/README.md#getentitybyid) - Get an entity by ID
* [UpdateEntityByID](docs/sdks/entities/README.md#updateentitybyid) - Update an entity by ID

### [Logs](docs/sdks/logs/README.md)

* [SearchLogs](docs/sdks/logs/README.md#searchlogs) - Search logs
* [ListLogArchives](docs/sdks/logs/README.md#listlogarchives) - Retrieve location and metadata of log archives

### [Metadata](docs/sdks/metadata/README.md)

* [ListEntityTypes](docs/sdks/metadata/README.md#listentitytypes) - List all entity types
* [ListMetricsForEntityType](docs/sdks/metadata/README.md#listmetricsforentitytype) - List metrics metadata for an entity type

### [Metrics](docs/sdks/metrics/README.md)

* [ListMetrics](docs/sdks/metrics/README.md#listmetrics) - List metrics
* [CreateCompositeMetric](docs/sdks/metrics/README.md#createcompositemetric) - Create composite metric
* [ListMultiMetricMeasurements](docs/sdks/metrics/README.md#listmultimetricmeasurements) - List measurements for a batch of metrics
* [UpdateCompositeMetric](docs/sdks/metrics/README.md#updatecompositemetric) - Update composite metric
* [DeleteCompositeMetric](docs/sdks/metrics/README.md#deletecompositemetric) - Delete composite metric
* [GetMetricByName](docs/sdks/metrics/README.md#getmetricbyname) - Get metric info by name
* [ListMetricAttributes](docs/sdks/metrics/README.md#listmetricattributes) - List metric attribute names
* [ListMetricAttributeValues](docs/sdks/metrics/README.md#listmetricattributevalues) - List metric attribute values
* [ListMetricMeasurements](docs/sdks/metrics/README.md#listmetricmeasurements) - List metric measurement values, grouped by attributes, filtered by the filter. An empty list indicates no data points are available for the given parameters.

### [Tokens](docs/sdks/tokens/README.md)

* [CreateToken](docs/sdks/tokens/README.md#createtoken) - Create ingestion token

</details>
<!-- End Available Resources and Operations [operations] -->

<!-- Start Pagination [pagination] -->
## Pagination

Some of the endpoints in this SDK support pagination. To use pagination, you make your SDK calls as usual, but the
returned response object will have a `Next` method that can be called to pull down the next group of results. If the
return value of `Next` is `nil`, then there are no more pages to be fetched.

Here's an example of one such pagination call:
```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/operations"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.Entities.ListEntities(ctx, operations.ListEntitiesRequest{
		Type: "<value>",
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		for {
			// handle items

			res, err = res.Next()

			if err != nil {
				// handle error
			}

			if res == nil {
				break
			}
		}
	}
}

```
<!-- End Pagination [pagination] -->

<!-- Start Retries [retries] -->
## Retries

Some of the endpoints in this SDK support retries. If you use the SDK without any configuration, it will fall back to the default retry strategy provided by the API. However, the default retry strategy can be overridden on a per-operation basis, or across the entire SDK.

To change the default retry strategy for a single API call, simply provide a `retry.Config` object to the call by using the `WithRetries` option:
```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"github.com/solarwinds/swo-sdk-go/swov1/retry"
	"log"
	"models/operations"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	}, operations.WithRetries(
		retry.Config{
			Strategy: "backoff",
			Backoff: &retry.BackoffStrategy{
				InitialInterval: 1,
				MaxInterval:     50,
				Exponent:        1.1,
				MaxElapsedTime:  100,
			},
			RetryConnectionErrors: false,
		}))
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```

If you'd like to override the default retry strategy for all operations that support retries, you can use the `WithRetryConfig` option at SDK initialization:
```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"github.com/solarwinds/swo-sdk-go/swov1/retry"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithRetryConfig(
			retry.Config{
				Strategy: "backoff",
				Backoff: &retry.BackoffStrategy{
					InitialInterval: 1,
					MaxInterval:     50,
					Exponent:        1.1,
					MaxElapsedTime:  100,
				},
				RetryConnectionErrors: false,
			}),
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations. All operations return a response object or an error, they will never return both.

By Default, an API error will return `apierrors.APIError`. When custom error responses are specified for an operation, the SDK may also return their associated error. You can refer to respective *Errors* tables in SDK docs for more details on possible error types for each operation.

For example, the `CreateChangeEvent` function may return the following errors:

| Error Type                                | Status Code | Content Type     |
| ----------------------------------------- | ----------- | ---------------- |
| apierrors.CommonBadRequestErrorResponse   | 400         | application/json |
| apierrors.CommonUnauthorizedErrorResponse | 401         | application/json |
| apierrors.CommonInternalErrorResponse     | 500         | application/json |
| apierrors.APIError                        | 4XX, 5XX    | \*/\*            |

### Example

```go
package main

import (
	"context"
	"errors"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/apierrors"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {

		var e *apierrors.CommonBadRequestErrorResponse
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}

		var e *apierrors.CommonUnauthorizedErrorResponse
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}

		var e *apierrors.CommonInternalErrorResponse
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}

		var e *apierrors.APIError
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}
	}
}

```
<!-- End Error Handling [errors] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Server Variables

The default server `https://api.na-01.cloud.solarwinds.com` contains variables and is set to `https://api.na-01.cloud.solarwinds.com` by default. To override default values, the following options are available when initializing the SDK client instance:

| Variable | Option                      | Default   | Description |
| -------- | --------------------------- | --------- | ----------- |
| `region` | `WithRegion(region string)` | `"na-01"` | Region name |

#### Example

```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithServerIndex(0),
		swov1.WithRegion("na-01"),
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```

### Override Server URL Per-Client

The default server can be overridden globally using the `WithServerURL(serverURL string)` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithServerURL("https://api.na-01.cloud.solarwinds.com"),
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEventsChangeEvent{
		ID:        swov1.Pointer[int64](1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Pointer[int64](1731676626),
		Source:    swov1.Pointer("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```
<!-- End Server Selection [server] -->

<!-- Start Custom HTTP Client [http-client] -->
## Custom HTTP Client

The Go SDK makes API calls that wrap an internal HTTP client. The requirements for the HTTP client are very simple. It must match this interface:

```go
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

The built-in `net/http` client satisfies this interface and a default client based on the built-in is provided by default. To replace this default with a client of your own, you can implement this interface yourself or provide your own client configured as desired. Here's a simple example, which adds a client with a 30 second timeout.

```go
import (
	"net/http"
	"time"

	"github.com/solarwinds/swo-sdk-go/swov1"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = swov1.New(swov1.WithClient(httpClient))
)
```

This can be a convenient way to configure timeouts, cookies, proxies, custom headers, and other low-level configuration.
<!-- End Custom HTTP Client [http-client] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This SDK is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this SDK, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation. 
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/solarwinds/swo-sdk-go/v1&utm_campaign=go)
