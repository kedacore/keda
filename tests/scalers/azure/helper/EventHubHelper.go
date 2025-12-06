//go:build e2e
// +build e2e

package helper

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/tests/helper"
)

type EventHubHelper struct {
	tenant            string
	subscription      string
	resourceGroup     string
	eventHubNamespace string
	eventHubName      string
	clientID          string
	clientSecret      string
	policyName        string
	connectionString  string
	producer          *azeventhubs.ProducerClient
}

func NewEventHubHelper(t *testing.T) EventHubHelper {
	require.NotEmpty(t, os.Getenv("TF_AZURE_SP_TENANT"), "TF_AZURE_SP_TENANT env variable is required for azure eventhub test")
	require.NotEmpty(t, os.Getenv("TF_AZURE_SUBSCRIPTION"), "TF_AZURE_SUBSCRIPTION env variable is required for azure eventhub test")
	require.NotEmpty(t, os.Getenv("TF_AZURE_RESOURCE_GROUP"), "TF_AZURE_RESOURCE_GROUP env variable is required for azure eventhub test")
	require.NotEmpty(t, os.Getenv("TF_AZURE_EVENTHUB_NAMESPACE"), "TF_AZURE_EVENTHUB_NAMESPACE env variable is required for azure eventhub test")
	require.NotEmpty(t, os.Getenv("TF_AZURE_SP_APP_ID"), "TF_AZURE_SP_APP_ID env variable is required for azure eventhub test")
	require.NotEmpty(t, os.Getenv("AZURE_SP_KEY"), "AZURE_SP_KEY env variable is required for azure eventhub test")

	randomNumber := helper.GetRandomNumber()
	return EventHubHelper{
		tenant:            os.Getenv("TF_AZURE_SP_TENANT"),
		subscription:      os.Getenv("TF_AZURE_SUBSCRIPTION"),
		resourceGroup:     os.Getenv("TF_AZURE_RESOURCE_GROUP"),
		eventHubNamespace: os.Getenv("TF_AZURE_EVENTHUB_NAMESPACE"),
		clientID:          os.Getenv("TF_AZURE_SP_APP_ID"),
		clientSecret:      os.Getenv("AZURE_SP_KEY"),
		policyName:        "e2e-tests",
		eventHubName:      fmt.Sprintf("keda-eh-%d", randomNumber),
	}
}

func (e *EventHubHelper) CreateEventHub(ctx context.Context, t *testing.T) {
	cred, err := azidentity.NewClientSecretCredential(e.tenant, e.clientID, e.clientSecret, nil)
	assert.NoErrorf(t, err, "cannot create azure credentials - %s", err)

	hubFactory, err := armeventhub.NewClientFactory(e.subscription, cred, nil)
	assert.NoErrorf(t, err, "cannot create azure arm factory - %s", err)

	hubClient := hubFactory.NewEventHubsClient()
	_, err = hubClient.CreateOrUpdate(ctx, e.resourceGroup, e.eventHubNamespace, e.eventHubName, armeventhub.Eventhub{
		Properties: &armeventhub.Properties{
			MessageRetentionInDays: to.Ptr[int64](1),
			PartitionCount:         to.Ptr[int64](1),
			Status:                 to.Ptr(armeventhub.EntityStatusActive),
			CaptureDescription:     nil,
		},
	}, &armeventhub.EventHubsClientCreateOrUpdateOptions{})
	assert.NoErrorf(t, err, "cannot create azure event hub - %s", err)

	_, err = hubClient.CreateOrUpdateAuthorizationRule(ctx, e.resourceGroup, e.eventHubNamespace, e.eventHubName, e.policyName, armeventhub.AuthorizationRule{
		Properties: &armeventhub.AuthorizationRuleProperties{
			Rights: []*armeventhub.AccessRights{
				to.Ptr(armeventhub.AccessRightsListen),
				to.Ptr(armeventhub.AccessRightsManage),
				to.Ptr(armeventhub.AccessRightsSend),
			},
		},
	}, nil)
	assert.NoErrorf(t, err, "cannot create azure event hub - %s", err)

	keys, err := hubClient.ListKeys(ctx, e.resourceGroup, e.eventHubNamespace, e.eventHubName, e.policyName, nil)
	assert.NoErrorf(t, err, "cannot get azure event hub keys- %s", err)
	e.connectionString = *keys.PrimaryConnectionString

	producer, err := azeventhubs.NewProducerClientFromConnectionString(e.connectionString, "", nil)
	e.producer = producer
	assert.NoErrorf(t, err, "cannot create event hub producer - %s", err)
}

func (e *EventHubHelper) DeleteEventHub(ctx context.Context, t *testing.T) {
	cred, err := azidentity.NewClientSecretCredential(e.tenant, e.clientID, e.clientSecret, nil)
	assert.NoErrorf(t, err, "cannot create azure credentials - %s", err)
	hubFactory, err := armeventhub.NewClientFactory(e.subscription, cred, nil)
	assert.NoErrorf(t, err, "cannot create azure arm factory - %s", err)
	hubClient := hubFactory.NewEventHubsClient()
	_, err = hubClient.Delete(ctx, e.resourceGroup, e.eventHubNamespace, e.eventHubName, nil)
	assert.NoErrorf(t, err, "cannot delete event hub - %s", err)
}

func (e *EventHubHelper) PublishEventHubdEvents(ctx context.Context, t *testing.T, count int) {
	batch, err := e.producer.NewEventDataBatch(ctx, nil)
	assert.NoErrorf(t, err, "cannot create the batch - %s", err)
	for i := 0; i < count; i++ {
		now := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
			now.Year(), now.Month(), now.Day(),
			now.Hour(), now.Minute(), now.Second())
		msg := fmt.Sprintf("Message - %s", formatted)
		err = batch.AddEventData(&azeventhubs.EventData{
			Body: []byte(msg),
		}, nil)
		assert.NoErrorf(t, err, "cannot batch the event - %s", err)
	}
	err = e.producer.SendEventDataBatch(ctx, batch, nil)
	assert.NoErrorf(t, err, "cannot send the batch - %s", err)
}

func (e *EventHubHelper) EventHubNamespace() string {
	return e.eventHubNamespace
}

func (e *EventHubHelper) EventHub() string {
	return e.eventHubName
}

func (e *EventHubHelper) ConnectionString() string {
	return e.connectionString
}
