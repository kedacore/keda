- [User Guide](#user-guide)
	- [Example](#example)
	- [Amazon OpenSearch Service](#amazon-opensearch-service)
		- [AWS SDK v1](#aws-sdk-v1)
		- [AWS SDK v2](#aws-sdk-v2)
	- [Guides by Topic](#guides-by-topic)

# User Guide

## Example

In the example below, we create a client, an index with non-default settings, insert a document to the index, search for the document, delete the document and finally delete the index.

```go
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchutil"
)

const IndexName = "go-test-index1"

func main() {
	if err := example(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func example() error {
	// Initialize the client with SSL/TLS enabled.
	client, err := opensearchapi.NewClient(
		opensearchapi.Config{
			Client: opensearch.Config{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // For testing only. Use certificate for validation.
				},
				Addresses: []string{"https://localhost:9200"},
				Username:  "admin", // For testing only. Don't store credentials in code.
				Password:  "myStrongPassword123!",
			},
		},
	)
	if err != nil {
		return err
	}

	ctx := context.Background()
	// Print OpenSearch version information on console.
	infoResp, err := client.Info(ctx, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Cluster INFO:\n  Cluster Name: %s\n  Cluster UUID: %s\n  Version Number: %s\n", infoResp.ClusterName, infoResp.ClusterUUID, infoResp.Version.Number)

	// Define index mapping.
	// Note: these particular settings (eg, shards/replicas)
	// will have no effect in AWS OpenSearch Serverless
	mapping := strings.NewReader(`{
	    "settings": {
	        "index": {
	            "number_of_shards": 4
	        }
	    }
	}`)

	// Create an index with non-default settings.
	createIndexResponse, err := client.Indices.Create(
		ctx,
		opensearchapi.IndicesCreateReq{
			Index: IndexName,
			Body:  mapping,
		},
	)

	var opensearchError *opensearch.StructError

	// Load err into opensearch.Error to access the fields and tolerate if the index already exists
	if err != nil {
		if errors.As(err, &opensearchError) {
			if opensearchError.Err.Type != "resource_already_exists_exception" {
				return err
			}
		} else {
			return err
		}
	}
	fmt.Printf("Created Index: %s\n  Shards Acknowledged: %t\n", createIndexResponse.Index, createIndexResponse.ShardsAcknowledged)

	// When using a structure, the conversion process to io.Reader can be omitted using utility functions.
	document := struct {
		Title    string `json:"title"`
		Director string `json:"director"`
		Year     string `json:"year"`
	}{
		Title:    "Moneyball",
		Director: "Bennett Miller",
		Year:     "2011",
	}

	docId := "1"
	insertResp, err := client.Index(
		ctx,
		opensearchapi.IndexReq{
			Index:      IndexName,
			DocumentID: docId,
			Body:       opensearchutil.NewJSONReader(&document),
			Params: opensearchapi.IndexParams{
				Refresh: "true",
			},
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Created document in %s\n  ID: %s\n", insertResp.Index, insertResp.ID)

	// Search for the document.
	content := strings.NewReader(`{
		"size": 5,
	    "query": {
	        "multi_match": {
	            "query": "miller",
	            "fields": ["title^2", "director"]
	        }
	    }
	}`)

	searchResp, err := client.Search(
		ctx,
		&opensearchapi.SearchReq{
			Body: content,
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Search hits: %v\n", searchResp.Hits.Total.Value)

	if searchResp.Hits.Total.Value > 0 {
		indices := make([]string, 0)
		for _, hit := range searchResp.Hits.Hits {
			add := true
			for _, index := range indices {
				if index == hit.Index {
					add = false
				}
			}
			if add {
				indices = append(indices, hit.Index)
			}
		}
		fmt.Printf("Search indices: %s\n", strings.Join(indices, ","))
	}

	// Delete the document.
	deleteReq := opensearchapi.DocumentDeleteReq{
		Index:      IndexName,
		DocumentID: docId,
	}

	deleteResponse, err := client.Document.Delete(ctx, deleteReq)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted document: %t\n", deleteResponse.Result == "deleted")

	// Delete previously created index.
	deleteIndex := opensearchapi.IndicesDeleteReq{Indices: []string{IndexName}}

	deleteIndexResp, err := client.Indices.Delete(ctx, deleteIndex)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted index: %t\n", deleteIndexResp.Acknowledged)

	// Try to delete the index again which fails as it does not exist
	_, err = client.Indices.Delete(ctx, deleteIndex)

	// Load err into opensearchapi.Error to access the fields and tolerate if the index is missing
	if err != nil {
		if errors.As(err, &opensearchError) {
			if opensearchError.Err.Type != "index_not_found_exception" {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

```

## Amazon OpenSearch Service

Before starting, we strongly recommend reading the full AWS documentation regarding using IAM credentials to sign requests to OpenSearch APIs. See [Identity and Access Management in Amazon OpenSearch Service.](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/ac.html)

> Even if you configure a completely open resource-based access policy, all requests to the OpenSearch Service configuration API must be signed. If your policies specify IAM users or roles, requests to the OpenSearch APIs also must be signed using AWS Signature Version 4.
>
> See [Managed Domains signing-service requests.](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/ac.html#managedomains-signing-service-requests)

Depending on the version of AWS SDK used, import the v1 or v2 request signer from `signer/aws` or `signer/awsv2` respectively. Both signers are equivalent in their functionality, they provide AWS Signature Version 4 (SigV4).

To read more about SigV4 see [Signature Version 4 signing process](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)

Here are some Go samples that show how to sign each OpenSearch request and automatically search for AWS credentials from the ~/.aws folder or environment variables:

### AWS SDK v1

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/aws"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

const IndexName = "go-test-index1"

func main() {
	if err := example(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

const endpoint = "" // e.g. https://opensearch-domain.region.com

func example() error {
	// Create an AWS request Signer and load AWS configuration using default config folder or env vars.
	// See https://docs.aws.amazon.com/opensearch-service/latest/developerguide/request-signing.html#request-signing-go
	signer, err := requestsigner.NewSignerWithService(
		session.Options{SharedConfigState: session.SharedConfigEnable},
		requestsigner.OpenSearchService, // Use requestsigner.OpenSearchServerless for Amazon OpenSearch Serverless.
	)
	if err != nil {
		return err
	}
	// Create an opensearch client and use the request-signer.
	client, err := opensearchapi.NewClient(
		opensearchapi.Config{
			Client: opensearch.Config{
				Addresses: []string{endpoint},
				Signer:    signer,
			},
		},
	)
	if err != nil {
		return err
	}

	ctx := context.Background()
    
    ping, err := client.Ping(ctx, nil)
    if err != nil {
        return err
    }

    fmt.Println(ping)

	return nil
}
```

### AWS SDK v2

Use the AWS SDK v2 for Go to authenticate with Amazon OpenSearch service.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

const endpoint = "" // e.g. https://opensearch-domain.region.com or Amazon OpenSearch Serverless endpoint

func main() {
	if err := example(); err != nil {
		fmt.Println(fmt.Sprintf("Error: %s", err))
		os.Exit(1)
	}
}
func example() error {
	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("<AWS_REGION>"),
		config.WithCredentialsProvider(
			getCredentialProvider("<AWS_ACCESS_KEY>", "<AWS_SECRET_ACCESS_KEY>", "<AWS_SESSION_TOKEN>"),
		),
	)
	if err != nil {
		return err
	}

	// Create an AWS request Signer and load AWS configuration using default config folder or env vars.
	signer, err := requestsigner.NewSignerWithService(awsCfg, "es") // Use "aoss" for Amazon OpenSearch Serverless
	if err != nil {
		return err
	}

	// Create an opensearch client and use the request-signer.
	client, err := opensearchapi.NewClient(
		opensearchapi.Config{
			Client: opensearch.Config{
				Addresses: []string{endpoint},
				Signer:    signer,
			},
		},
	)
	if err != nil {
		return err
	}

	indexName := "go-test-index"

	// Define index mapping.
	mapping := strings.NewReader(`{
	 "settings": {
	   "index": {
	        "number_of_shards": 4
	        }
	      }
	 }`)

	// Create an index with non-default settings.
	createResp, err := client.Indices.Create(
		ctx,
		opensearchapi.IndicesCreateReq{
			Index: indexName,
			Body:  mapping,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("created index: %s\n", createResp.Index)

	delResp, err := client.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{Indices: []string{indexName}})
	if err != nil {
		return err
	}

	fmt.Printf("deleted index: %#v\n", delResp.Acknowledged)
	return nil
}

func getCredentialProvider(accessKey, secretAccessKey, token string) aws.CredentialsProviderFunc {
	return func(ctx context.Context) (aws.Credentials, error) {
		c := &aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretAccessKey,
			SessionToken:    token,
		}
		return *c, nil
	}
}
```

## Guides by Topic

- [Index Lifecycle](guides/index_lifecycle.md)
- [Document Lifecycle](guides/document_lifecycle.md)
- [Search](guides/search.md)
- [Bulk](guides/bulk.md)
- [Advanced Index Actions](guides/advanced_index_actions.md)
- [Index Templates](guides/index_template.md)
- [Data Streams](guides/data_streams.md)
- [Retry and Backoff](guides/retry_backoff.md)
