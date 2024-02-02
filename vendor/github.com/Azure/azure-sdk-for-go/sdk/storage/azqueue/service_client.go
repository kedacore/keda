//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azqueue

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/base"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/shared"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/queueerror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/sas"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientOptions contains the optional parameters when creating a ServiceClient or QueueClient.
type ClientOptions struct {
	azcore.ClientOptions
}

// ServiceClient represents a URL to the Azure Queue Storage service allowing you to manipulate queues.
type ServiceClient base.Client[generated.ServiceClient]

// NewServiceClient creates an instance of ServiceClient with the specified values.
//   - serviceURL - the URL of the storage account e.g. https://<account>.queue.core.windows.net/
//   - cred - an Azure AD credential, typically obtained via the azidentity module
//   - options - client options; pass nil to accept the default values
func NewServiceClient(serviceURL string, cred azcore.TokenCredential, options *ClientOptions) (*ServiceClient, error) {
	authPolicy := runtime.NewBearerTokenPolicy(cred, []string{shared.TokenScope}, nil)
	conOptions := shared.GetClientOptions(options)
	conOptions.PerRetryPolicies = append(conOptions.PerRetryPolicies, authPolicy)
	pl := runtime.NewPipeline(exported.ModuleName, exported.ModuleVersion, runtime.PipelineOptions{}, &conOptions.ClientOptions)

	return (*ServiceClient)(base.NewServiceClient(serviceURL, pl, nil)), nil
}

// NewServiceClientWithNoCredential creates an instance of ServiceClient with the specified values.
// This is used to anonymously access a storage account or with a shared access signature (SAS) token.
//   - serviceURL - the URL of the storage account e.g. https://<account>.queue.core.windows.net/?<sas token>
//   - options - client options; pass nil to accept the default values
func NewServiceClientWithNoCredential(serviceURL string, options *ClientOptions) (*ServiceClient, error) {
	conOptions := shared.GetClientOptions(options)
	pl := runtime.NewPipeline(exported.ModuleName, exported.ModuleVersion, runtime.PipelineOptions{}, &conOptions.ClientOptions)

	return (*ServiceClient)(base.NewServiceClient(serviceURL, pl, nil)), nil
}

// NewServiceClientWithSharedKeyCredential creates an instance of ServiceClient with the specified values.
//   - serviceURL - the URL of the storage account e.g. https://<account>.queue.core.windows.net/
//   - cred - a SharedKeyCredential created with the matching storage account and access key
//   - options - client options; pass nil to accept the default values
func NewServiceClientWithSharedKeyCredential(serviceURL string, cred *SharedKeyCredential, options *ClientOptions) (*ServiceClient, error) {
	authPolicy := exported.NewSharedKeyCredPolicy(cred)
	conOptions := shared.GetClientOptions(options)
	conOptions.PerRetryPolicies = append(conOptions.PerRetryPolicies, authPolicy)
	pl := runtime.NewPipeline(exported.ModuleName, exported.ModuleVersion, runtime.PipelineOptions{}, &conOptions.ClientOptions)

	return (*ServiceClient)(base.NewServiceClient(serviceURL, pl, cred)), nil
}

// NewServiceClientFromConnectionString creates an instance of ServiceClient with the specified values.
//   - connectionString - a connection string for the desired storage account
//   - options - client options; pass nil to accept the default values
func NewServiceClientFromConnectionString(connectionString string, options *ClientOptions) (*ServiceClient, error) {
	parsed, err := shared.ParseConnectionString(connectionString)
	if err != nil {
		return nil, err
	}

	if parsed.AccountKey != "" && parsed.AccountName != "" {
		credential, err := exported.NewSharedKeyCredential(parsed.AccountName, parsed.AccountKey)
		if err != nil {
			return nil, err
		}
		return NewServiceClientWithSharedKeyCredential(parsed.ServiceURL, credential, options)
	}

	return NewServiceClientWithNoCredential(parsed.ServiceURL, options)
}

func (s *ServiceClient) generated() *generated.ServiceClient {
	return base.InnerClient((*base.Client[generated.ServiceClient])(s))
}

func (s *ServiceClient) sharedKey() *SharedKeyCredential {
	return base.SharedKey((*base.Client[generated.ServiceClient])(s))
}

// URL returns the URL endpoint used by the ServiceClient object.
func (s *ServiceClient) URL() string {
	return s.generated().Endpoint()
}

// GetServiceProperties - gets the properties of a storage account's Queue service, including properties for Storage Analytics
// and CORS (Cross-Origin Resource Sharing) rules.
func (s *ServiceClient) GetServiceProperties(ctx context.Context, o *GetServicePropertiesOptions) (GetServicePropertiesResponse, error) {
	getPropertiesOptions := o.format()
	resp, err := s.generated().GetProperties(ctx, getPropertiesOptions)
	return resp, err
}

// SetProperties Sets the properties of a storage account's Queue service, including Azure Storage Analytics.
// If an element (e.g. analytics_logging) is left as None, the existing settings on the service for that functionality are preserved.
func (s *ServiceClient) SetProperties(ctx context.Context, o *SetPropertiesOptions) (SetPropertiesResponse, error) {
	properties, setPropertiesOptions := o.format()
	resp, err := s.generated().SetProperties(ctx, properties, setPropertiesOptions)
	return resp, err
}

// GetStatistics Retrieves statistics related to replication for the Queue service.
func (s *ServiceClient) GetStatistics(ctx context.Context, o *GetStatisticsOptions) (GetStatisticsResponse, error) {
	getStatisticsOptions := o.format()
	resp, err := s.generated().GetStatistics(ctx, getStatisticsOptions)

	return resp, err
}

// NewListQueuesPager operation returns a pager of the queues under the specified account.
// Use an empty Marker to start enumeration from the beginning. Queue names are returned in lexicographic order.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/list-queues1.
func (s *ServiceClient) NewListQueuesPager(o *ListQueuesOptions) *runtime.Pager[ListQueuesResponse] {
	listOptions := generated.ServiceClientListQueuesSegmentOptions{}
	if o != nil {
		if o.Include.Metadata {
			listOptions.Include = append(listOptions.Include, "metadata")
		}
		listOptions.Marker = o.Marker
		listOptions.Maxresults = o.MaxResults
		listOptions.Prefix = o.Prefix
	}
	return runtime.NewPager(runtime.PagingHandler[ListQueuesResponse]{
		More: func(page ListQueuesResponse) bool {
			return page.NextMarker != nil && len(*page.NextMarker) > 0
		},
		Fetcher: func(ctx context.Context, page *ListQueuesResponse) (ListQueuesResponse, error) {
			var req *policy.Request
			var err error
			if page == nil {
				req, err = s.generated().ListQueuesSegmentCreateRequest(ctx, &listOptions)
			} else {
				listOptions.Marker = page.NextMarker
				req, err = s.generated().ListQueuesSegmentCreateRequest(ctx, &listOptions)
			}
			if err != nil {
				return ListQueuesResponse{}, err
			}
			resp, err := s.generated().Pipeline().Do(req)
			if err != nil {
				return ListQueuesResponse{}, err
			}
			if !runtime.HasStatusCode(resp, http.StatusOK) {
				return ListQueuesResponse{}, runtime.NewResponseError(resp)
			}
			return s.generated().ListQueuesSegmentHandleResponse(resp)
		},
	})
}

// NewQueueClient creates a new QueueClient object by concatenating queueName to the end of
// this Client's URL. The new QueueClient uses the same request policy pipeline as the Client.
func (s *ServiceClient) NewQueueClient(queueName string) *QueueClient {
	queueName = url.PathEscape(queueName)
	queueURL := runtime.JoinPaths(s.URL(), queueName)
	return (*QueueClient)(base.NewQueueClient(queueURL, s.generated().Pipeline(), s.sharedKey()))
}

// CreateQueue creates a new queue within a storage account. If a queue with the same name already exists, the operation fails.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/create-queue4.
func (s *ServiceClient) CreateQueue(ctx context.Context, queueName string, options *CreateOptions) (CreateResponse, error) {
	queueName = url.PathEscape(queueName)
	queueURL := runtime.JoinPaths(s.URL(), queueName)
	qC := (*QueueClient)(base.NewQueueClient(queueURL, s.generated().Pipeline(), s.sharedKey()))
	return qC.Create(ctx, options)
}

// DeleteQueue deletes the specified queue.
// For more information, see https://learn.microsoft.com/en-us/rest/api/storageservices/delete-queue3.
func (s *ServiceClient) DeleteQueue(ctx context.Context, queueName string, options *DeleteOptions) (DeleteResponse, error) {
	queueName = url.PathEscape(queueName)
	queueURL := runtime.JoinPaths(s.URL(), queueName)
	qC := (*QueueClient)(base.NewQueueClient(queueURL, s.generated().Pipeline(), s.sharedKey()))
	return qC.Delete(ctx, options)
}

// GetSASURL is a convenience method for generating a SAS token for the currently pointed at account.
// It can only be used if the credential supplied during creation was a SharedKeyCredential.
// This validity can be checked with CanGetAccountSASToken().
func (s *ServiceClient) GetSASURL(resources sas.AccountResourceTypes, permissions sas.AccountPermissions, expiry time.Time, o *GetSASURLOptions) (string, error) {
	if s.sharedKey() == nil {
		return "", queueerror.MissingSharedKeyCredential
	}
	st := o.format()
	qps, err := sas.AccountSignatureValues{
		Version:       sas.Version,
		Protocol:      sas.ProtocolHTTPS,
		Permissions:   permissions.String(),
		ResourceTypes: resources.String(),
		StartTime:     st,
		ExpiryTime:    expiry.UTC(),
	}.SignWithSharedKey(s.sharedKey())
	if err != nil {
		return "", err
	}

	endpoint := s.URL()
	if !strings.HasSuffix(endpoint, "/") {
		// add a trailing slash to be consistent with the portal
		endpoint += "/"
	}
	endpoint += "?" + qps.Encode()

	return endpoint, nil
}
