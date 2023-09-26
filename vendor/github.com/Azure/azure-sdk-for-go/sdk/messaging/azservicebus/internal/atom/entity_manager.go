// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package atom

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/conn"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/sbauth"
)

const (
	serviceBusSchema = "http://schemas.microsoft.com/netservices/2010/10/servicebus/connect"
	atomSchema       = "http://www.w3.org/2005/Atom"
	applicationXML   = "application/xml"
)

type (
	EntityManager interface {
		Get(ctx context.Context, entityPath string, respObj any) (*http.Response, error)
		Put(ctx context.Context, entityPath string, body any, respObj any, options *ExecuteOptions) (*http.Response, error)
		Delete(ctx context.Context, entityPath string) (*http.Response, error)
		TokenProvider() auth.TokenProvider
	}

	// entityManager provides CRUD functionality for Service Bus entities (Queues, Topics, Subscriptions...)
	entityManager struct {
		tokenProvider auth.TokenProvider
		Host          string
		pl            runtime.Pipeline
	}

	// BaseEntityDescription provides common fields which are part of Queues, Topics and Subscriptions
	BaseEntityDescription struct {
		InstanceMetadataSchema *string `xml:"xmlns:i,attr,omitempty"`
		ServiceBusSchema       *string `xml:"xmlns,attr,omitempty"`
	}

	// example: <Error><Code>401</Code><Detail>Manage,EntityRead claims required for this operation.</Detail></Error>
	ManagementError struct {
		XMLName xml.Name `xml:"Error"`
		Code    int      `xml:"Code"`
		Detail  string   `xml:"Detail"`
	}

	// CountDetails has current active (and other) messages for queue/topic.
	CountDetails struct {
		XMLName                        xml.Name `xml:"CountDetails"`
		ActiveMessageCount             *int32   `xml:"ActiveMessageCount,omitempty"`
		DeadLetterMessageCount         *int32   `xml:"DeadLetterMessageCount,omitempty"`
		ScheduledMessageCount          *int32   `xml:"ScheduledMessageCount,omitempty"`
		TransferDeadLetterMessageCount *int32   `xml:"TransferDeadLetterMessageCount,omitempty"`
		TransferMessageCount           *int32   `xml:"TransferMessageCount,omitempty"`
	}

	// EntityStatus enumerates the values for entity status.
	EntityStatus string
)

const (
	// Active ...
	Active EntityStatus = "Active"
	// Creating ...
	Creating EntityStatus = "Creating"
	// Deleting ...
	Deleting EntityStatus = "Deleting"
	// Disabled ...
	Disabled EntityStatus = "Disabled"
	// ReceiveDisabled ...
	ReceiveDisabled EntityStatus = "ReceiveDisabled"
	// Renaming ...
	Renaming EntityStatus = "Renaming"
	// Restoring ...
	Restoring EntityStatus = "Restoring"
	// SendDisabled ...
	SendDisabled EntityStatus = "SendDisabled"
	// Unknown ...
	Unknown EntityStatus = "Unknown"
)

func (m *ManagementError) String() string {
	return fmt.Sprintf("Code: %d, Details: %s", m.Code, m.Detail)
}

// NewEntityManagerWithConnectionString creates an entity manager (a lower level HTTP client
// for the ATOM endpoint). This is typically wrapped by an entity specific client (like
// TopicManager, QueueManager or , SubscriptionManager).
func NewEntityManagerWithConnectionString(connectionString string, version string, options *azcore.ClientOptions) (EntityManager, error) {
	parsed, err := conn.ParsedConnectionFromStr(connectionString)

	if err != nil {
		return nil, err
	}

	provider, err := sbauth.NewTokenProviderWithConnectionString(parsed)

	if err != nil {
		return nil, err
	}

	return newEntityManagerImpl(provider, version, options, parsed.Namespace)
}

// NewEntityManager creates an entity manager using a TokenCredential.
func NewEntityManager(ns string, tokenCredential azcore.TokenCredential, version string, options *azcore.ClientOptions) (EntityManager, error) {
	provider := sbauth.NewTokenProvider(tokenCredential)
	return newEntityManagerImpl(provider, version, options, ns)
}

// Get performs an HTTP Get for a given entity path, deserializing the returned XML into `respObj`
func (em *entityManager) Get(ctx context.Context, entityPath string, respObj any) (*http.Response, error) {
	resp, err := em.execute(ctx, http.MethodGet, entityPath, nil, nil)
	defer CloseRes(ctx, resp)

	if err != nil {
		return resp, err
	}

	return deserializeBody(resp, respObj)
}

// Put performs an HTTP PUT for a given entity path and body, deserializing the returned XML into `respObj`
func (em *entityManager) Put(ctx context.Context, entityPath string, body any, respObj any, options *ExecuteOptions) (*http.Response, error) {
	bodyBytes, err := xml.Marshal(body)

	if err != nil {
		return nil, err
	}

	resp, err := em.execute(ctx, http.MethodPut, entityPath, bytes.NewReader(bodyBytes), options)
	defer CloseRes(ctx, resp)

	if err != nil {
		return resp, err
	}

	return deserializeBody(resp, respObj)
}

// Delete performs an HTTP DELETE for a given entity path
func (em *entityManager) Delete(ctx context.Context, entityPath string) (*http.Response, error) {
	return em.execute(ctx, http.MethodDelete, entityPath, nil, nil)
}

type ExecuteOptions struct {
	ForwardTo           *string
	ForwardToDeadLetter *string
}

func (em *entityManager) execute(ctx context.Context, method string, entityPath string, body io.ReadSeeker, options *ExecuteOptions) (*http.Response, error) {
	url := em.Host + strings.TrimPrefix(entityPath, "/")

	ctx = context.WithValue(ctx, ctxWithAuthKey{}, options)

	req, err := runtime.NewRequest(ctx, method, url)

	if err != nil {
		return nil, err
	}

	q := req.Raw().URL.Query()
	q.Add("api-version", "2021-05")
	req.Raw().URL.RawQuery = q.Encode()

	if body != nil {
		if err := req.SetBody(streaming.NopCloser(body), "application/atom+xml;type=entry;charset=utf-8"); err != nil {
			return nil, err
		}
	}

	resp, err := em.pl.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, runtime.NewResponseError(resp)
	}

	return resp, nil
}

// TokenProvider generates authorization tokens for communicating with the Service Bus management API
func (em *entityManager) TokenProvider() auth.TokenProvider {
	return em.tokenProvider
}

func FormatManagementError(body []byte, origErr error) error {
	var mgmtError ManagementError
	unmarshalErr := xml.Unmarshal(body, &mgmtError)
	if unmarshalErr != nil {
		return origErr
	}

	return fmt.Errorf("error code: %d, Details: %s", mgmtError.Code, mgmtError.Detail)
}

var ErrFeedEmpty = errors.New("entity does not exist")

// deserializeBody deserializes the body of the response into the type specified by respObj
// (similar to xml.Unmarshal, which this func is calling).
// If an empty feed is found, it returns nil.
func deserializeBody(resp *http.Response, respObj any) (*http.Response, error) {
	bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return resp, err
	}

	if err := xml.Unmarshal(bytes, respObj); err != nil {
		// In ATOM when you request a specific entity (queue, topic, sub) you typically get an
		// <Entry>. However, if the entity is not found, instead of getting a 404 you actually
		// get a <Feed> XML object that is empty and an HTTP status code of 200.
		//
		// So the combination of "can't deserialize object" and "it's an empty feed" are enough
		// for us to note that we weren't expecting a feed (ie, GET /queue) and that the feed
		// itself is the special "empty feed".
		var emptyFeed QueueFeed
		feedErr := xml.Unmarshal(bytes, &emptyFeed)

		if feedErr == nil && emptyFeed.Title == "Publicly Listed Services" {
			return resp, ErrFeedEmpty
		}

		return resp, err
	}

	return resp, nil
}

func newEntityManagerImpl(provider *sbauth.TokenProvider, version string, options *policy.ClientOptions, ns string) (EntityManager, error) {
	popts := runtime.PipelineOptions{
		PerRetry: []policy.Policy{
			&perRetryAuthPolicy{tp: provider},
		},
	}

	pl := runtime.NewPipeline("azsbadmin", version, popts, options)

	return &entityManager{
		Host:          fmt.Sprintf("https://%s/", ns),
		tokenProvider: provider,
		pl:            pl,
	}, nil
}
