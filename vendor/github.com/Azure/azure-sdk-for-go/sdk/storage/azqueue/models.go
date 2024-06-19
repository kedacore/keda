//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azqueue

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/sas"
	"time"
)

// SharedKeyCredential contains an account's name and its primary or secondary key.
type SharedKeyCredential = exported.SharedKeyCredential

// NewSharedKeyCredential creates an immutable SharedKeyCredential containing the
// storage account's name and either its primary or secondary key.
func NewSharedKeyCredential(accountName, accountKey string) (*SharedKeyCredential, error) {
	return exported.NewSharedKeyCredential(accountName, accountKey)
}

// URLParts object represents the components that make up an Azure Storage Queue URL.
// NOTE: Changing any SAS-related field requires computing a new SAS signature.
type URLParts = sas.URLParts

// ParseURL parses a URL initializing URLParts' fields including any SAS-related & snapshot query parameters. Any other
// query parameters remain in the UnparsedParams field. This method overwrites all fields in the URLParts object.
func ParseURL(u string) (URLParts, error) {
	return sas.ParseURL(u)
}

// ================================================================

// CORSRule - CORS is an HTTP feature that enables a web application running under one domain to access resources in another
// domain. Web browsers implement a security restriction known as same-origin policy that
// prevents a web page from calling APIs in a different domain; CORS provides a secure way to allow one domain (the origin
// domain) to call APIs in another domain
type CORSRule = generated.CORSRule

// GeoReplication - Geo-Replication information for the Secondary Storage Service
type GeoReplication = generated.GeoReplication

// RetentionPolicy - the retention policy which determines how long the associated data should persist
type RetentionPolicy = generated.RetentionPolicy

// Metrics - a summary of request statistics grouped by API in hour or minute aggregates for queues
type Metrics = generated.Metrics

// Logging - Azure Analytics Logging settings.
type Logging = generated.Logging

// StorageServiceProperties - Storage Service Properties.
type StorageServiceProperties = generated.StorageServiceProperties

// StorageServiceStats - Stats for the storage service.
type StorageServiceStats = generated.StorageServiceStats

// SignedIdentifier - signed identifier
type SignedIdentifier = generated.SignedIdentifier

// EnqueuedMessage - enqueued message
type EnqueuedMessage = generated.EnqueuedMessage

// DequeuedMessage - dequeued message
type DequeuedMessage = generated.DequeuedMessage

// PeekedMessage - peeked message
type PeekedMessage = generated.PeekedMessage

// ListQueuesSegmentResponse - response segment
type ListQueuesSegmentResponse = generated.ListQueuesSegmentResponse

// Queue - queue item
type Queue = generated.Queue

// AccessPolicy - An Access policy
type AccessPolicy = generated.AccessPolicy

// AccessPolicyPermission type simplifies creating the permissions string for a queue's access policy.
// Initialize an instance of this type and then call its String method to set AccessPolicy's Permission field.
type AccessPolicyPermission = exported.AccessPolicyPermission

// ---------------------------------------------------------------------------------------------------------------------

// ListQueuesOptions provides set of configurations for ListQueues operation
type ListQueuesOptions struct {
	Include ListQueuesInclude

	// A string value that identifies the portion of the list of queues to be returned with the next listing operation. The
	// operation returns the NextMarker value within the response body if the listing operation did not return all queues
	// remaining to be listed with the current page. The NextMarker value can be used as the value for the marker parameter in
	// a subsequent call to request the next page of list items. The marker value is opaque to the client.
	Marker *string

	// Specifies the maximum number of queues to return. If the request does not specify max results, or specifies a value
	// greater than 5000, the server will return up to 5000 items. Note that if the listing operation crosses a partition boundary,
	// then the service will return a continuation token for retrieving the remainder of the results. For this reason, it is possible
	// that the service will return fewer results than specified by max results, or than the default of 5000.
	MaxResults *int32

	// Filters the results to return only queues whose name begins with the specified prefix.
	Prefix *string
}

// ListQueuesInclude indicates what additional information the service should return with each queue.
type ListQueuesInclude struct {
	// Tells the service whether to return metadata for each queue.
	Metadata bool
}

// ---------------------------------------------------------------------------------------------------------------------

// SetPropertiesOptions provides set of options for ServiceClient.SetProperties
type SetPropertiesOptions struct {
	// The set of CORS rules.
	CORS []*CORSRule

	// a summary of request statistics grouped by API in hour or minute aggregates for queues
	HourMetrics *Metrics

	// Azure Analytics Logging settings.
	Logging *Logging

	// a summary of request statistics grouped by API in hour or minute aggregates for queues
	MinuteMetrics *Metrics
}

func (o *SetPropertiesOptions) format() (generated.StorageServiceProperties, *generated.ServiceClientSetPropertiesOptions) {
	if o == nil {
		return generated.StorageServiceProperties{}, nil
	}

	defaultVersion := to.Ptr[string]("1.0")
	defaultAge := to.Ptr[int32](0)
	emptyStr := to.Ptr[string]("")

	if o.CORS != nil {
		for i := 0; i < len(o.CORS); i++ {
			if o.CORS[i].AllowedHeaders == nil {
				o.CORS[i].AllowedHeaders = emptyStr
			}
			if o.CORS[i].ExposedHeaders == nil {
				o.CORS[i].ExposedHeaders = emptyStr
			}
			if o.CORS[i].MaxAgeInSeconds == nil {
				o.CORS[i].MaxAgeInSeconds = defaultAge
			}
		}
	}

	if o.HourMetrics != nil {
		if o.HourMetrics.Version == nil {
			o.HourMetrics.Version = defaultVersion
		}
	}

	if o.Logging != nil {
		if o.Logging.Version == nil {
			o.Logging.Version = defaultVersion
		}
	}

	if o.MinuteMetrics != nil {
		if o.MinuteMetrics.Version == nil {
			o.MinuteMetrics.Version = defaultVersion
		}

	}

	return generated.StorageServiceProperties{
		CORS:          o.CORS,
		HourMetrics:   o.HourMetrics,
		Logging:       o.Logging,
		MinuteMetrics: o.MinuteMetrics,
	}, nil
}

// ---------------------------------------------------------------------------------------------------------------------

// GetServicePropertiesOptions contains the optional parameters for the ServiceClient.GetServiceProperties method.
type GetServicePropertiesOptions struct {
	// placeholder for future options
}

func (o *GetServicePropertiesOptions) format() *generated.ServiceClientGetPropertiesOptions {
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// GetStatisticsOptions provides set of options for ServiceClient.GetStatistics
type GetStatisticsOptions struct {
	// placeholder for future options
}

func (o *GetStatisticsOptions) format() *generated.ServiceClientGetStatisticsOptions {
	return nil
}

// -------------------------------------------------QUEUES--------------------------------------------------------------

// CreateOptions contains the optional parameters for creating a queue.
type CreateOptions struct {
	// Optional. Specifies a user-defined name-value pair associated with the queue.
	Metadata map[string]*string
}

func (o *CreateOptions) format() *generated.QueueClientCreateOptions {
	if o == nil {
		return nil
	}
	return &generated.QueueClientCreateOptions{Metadata: o.Metadata}
}

// ---------------------------------------------------------------------------------------------------------------------

// DeleteOptions contains the optional parameters for deleting a queue.
type DeleteOptions struct {
}

func (o *DeleteOptions) format() *generated.QueueClientDeleteOptions {
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// SetMetadataOptions contains the optional parameters for the QueueClient.SetMetadata method.
type SetMetadataOptions struct {
	Metadata map[string]*string
}

func (o *SetMetadataOptions) format() *generated.QueueClientSetMetadataOptions {
	if o == nil {
		return nil
	}

	return &generated.QueueClientSetMetadataOptions{Metadata: o.Metadata}
}

// ---------------------------------------------------------------------------------------------------------------------

// GetAccessPolicyOptions contains the optional parameters for the QueueClient.GetAccessPolicy method.
type GetAccessPolicyOptions struct {
}

func (o *GetAccessPolicyOptions) format() *generated.QueueClientGetAccessPolicyOptions {
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// SetAccessPolicyOptions provides set of configurations for QueueClient.SetAccessPolicy operation
type SetAccessPolicyOptions struct {
	QueueACL []*SignedIdentifier
}

func (o *SetAccessPolicyOptions) format() (*generated.QueueClientSetAccessPolicyOptions, []*SignedIdentifier, error) {
	if o == nil {
		return nil, nil, nil
	}
	if o.QueueACL != nil {
		for _, c := range o.QueueACL {
			err := formatTime(c)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return &generated.QueueClientSetAccessPolicyOptions{}, o.QueueACL, nil
}

func formatTime(c *SignedIdentifier) error {
	if c.AccessPolicy == nil {
		return nil
	}

	if c.AccessPolicy.Start != nil {
		st, err := time.Parse(time.RFC3339, c.AccessPolicy.Start.UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
		c.AccessPolicy.Start = &st
	}
	if c.AccessPolicy.Expiry != nil {
		et, err := time.Parse(time.RFC3339, c.AccessPolicy.Expiry.UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
		c.AccessPolicy.Expiry = &et
	}

	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// GetQueuePropertiesOptions contains the optional parameters for the QueueClient.GetProperties method.
type GetQueuePropertiesOptions struct {
}

func (o *GetQueuePropertiesOptions) format() *generated.QueueClientGetPropertiesOptions {
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// EnqueueMessageOptions contains the optional parameters for the QueueClient.EnqueueMessage method.
type EnqueueMessageOptions struct {
	// Specifies the time-to-live interval for the message, in seconds.
	// The time-to-live may be any positive number or -1 for infinity.
	// If this parameter is omitted, the default time-to-live is 7 days.
	TimeToLive *int32
	// If not specified, the default value is 0.
	// Specifies the new visibility timeout value, in seconds, relative to server time.
	// The value must be larger than or equal to 0, and cannot be larger than 7 days.
	// The visibility timeout of a message cannot be set to a value later than the expiry time.
	// VisibilityTimeout should be set to a value smaller than the time-to-live value.
	VisibilityTimeout *int32
}

func (o *EnqueueMessageOptions) format() *generated.MessagesClientEnqueueOptions {
	if o == nil {
		return nil
	}

	return &generated.MessagesClientEnqueueOptions{MessageTimeToLive: o.TimeToLive,
		Visibilitytimeout: o.VisibilityTimeout}
}

// ---------------------------------------------------------------------------------------------------------------------

// DequeueMessageOptions contains the optional parameters for the QueueClient.DequeueMessage method.
type DequeueMessageOptions struct {
	// If not specified, the default value is 0. Specifies the new visibility timeout value,
	// in seconds, relative to server time. The value must be larger than or equal to 0, and cannot be
	// larger than 7 days. The visibility timeout of a message cannot be
	// set to a value later than the expiry time. VisibilityTimeout
	// should be set to a value smaller than the time-to-live value.
	VisibilityTimeout *int32
}

func (o *DequeueMessageOptions) format() *generated.MessagesClientDequeueOptions {
	numberOfMessages := int32(1)
	if o == nil {
		return &generated.MessagesClientDequeueOptions{NumberOfMessages: &numberOfMessages}
	}

	return &generated.MessagesClientDequeueOptions{NumberOfMessages: &numberOfMessages,
		Visibilitytimeout: o.VisibilityTimeout}
}

// ---------------------------------------------------------------------------------------------------------------------

// DequeueMessagesOptions contains the optional parameters for the QueueClient.DequeueMessages method.
type DequeueMessagesOptions struct {
	// Optional. A nonzero integer value that specifies the number of messages to retrieve from the queue,
	// up to a maximum of 32. If fewer messages are visible, the visible messages are returned.
	// By default, a single message is retrieved from the queue with this operation.
	NumberOfMessages *int32
	// If not specified, the default value is 30. Specifies the
	// new visibility timeout value, in seconds, relative to server time.
	// The value must be larger than or equal to 1, and cannot be
	// larger than 7 days. The visibility timeout of a message cannot be
	// set to a value later than the expiry time. VisibilityTimeout
	// should be set to a value smaller than the time-to-live value.
	VisibilityTimeout *int32
}

func (o *DequeueMessagesOptions) format() *generated.MessagesClientDequeueOptions {
	if o == nil {
		return nil
	}

	return &generated.MessagesClientDequeueOptions{NumberOfMessages: o.NumberOfMessages,
		Visibilitytimeout: o.VisibilityTimeout}
}

// ---------------------------------------------------------------------------------------------------------------------

// UpdateMessageOptions contains the optional parameters for the QueueClient.UpdateMessage method.
type UpdateMessageOptions struct {
	VisibilityTimeout *int32
}

func (o *UpdateMessageOptions) format() *generated.MessageIDClientUpdateOptions {
	defaultVT := to.Ptr(int32(0))
	if o == nil {
		return &generated.MessageIDClientUpdateOptions{Visibilitytimeout: defaultVT}
	}
	if o.VisibilityTimeout == nil {
		o.VisibilityTimeout = defaultVT
	}
	return &generated.MessageIDClientUpdateOptions{Visibilitytimeout: o.VisibilityTimeout}
}

// ---------------------------------------------------------------------------------------------------------------------

// DeleteMessageOptions contains the optional parameters for the QueueClient.DeleteMessage method.
type DeleteMessageOptions struct {
}

func (o *DeleteMessageOptions) format() *generated.MessageIDClientDeleteOptions {
	if o == nil {
		return nil
	}

	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// PeekMessageOptions contains the optional parameters for the QueueClient.PeekMessage method.
type PeekMessageOptions struct {
}

func (o *PeekMessageOptions) format() *generated.MessagesClientPeekOptions {
	numberOfMessages := int32(1)
	return &generated.MessagesClientPeekOptions{NumberOfMessages: &numberOfMessages}
}

// ---------------------------------------------------------------------------------------------------------------------

// PeekMessagesOptions contains the optional parameters for the QueueClient.PeekMessages method.
type PeekMessagesOptions struct {
	NumberOfMessages *int32
}

func (o *PeekMessagesOptions) format() *generated.MessagesClientPeekOptions {
	if o == nil {
		return nil
	}

	return &generated.MessagesClientPeekOptions{NumberOfMessages: o.NumberOfMessages}
}

// ---------------------------------------------------------------------------------------------------------------------

// ClearMessagesOptions contains the optional parameters for the QueueClient.ClearMessages method.
type ClearMessagesOptions struct {
}

func (o *ClearMessagesOptions) format() *generated.MessagesClientClearOptions {
	if o == nil {
		return nil
	}

	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

// GetSASURLOptions contains the optional parameters for the Client.GetSASURL method.
type GetSASURLOptions struct {
	StartTime *time.Time
}

func (o *GetSASURLOptions) format() time.Time {
	if o == nil {
		return time.Time{}
	}
	var st time.Time
	if o.StartTime != nil {
		st = o.StartTime.UTC()
	}
	return st
}
