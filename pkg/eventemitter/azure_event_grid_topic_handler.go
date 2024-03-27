/*
Copyright 2024 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// ******************************* DESCRIPTION ****************************** \\
// AzureEventGridHandler focuses on emitting the CloudEventSource to Azure Event Grid
// ************************************************************************** \\

package eventemitter

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid/publisher"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
)

type AzureEventGridTopicHandler struct {
	Context      context.Context
	Endpoint     string
	Key          string
	ClusterName  string
	Client       *publisher.Client
	logger       logr.Logger
	activeStatus metav1.ConditionStatus
}

func NewAzureEventGridTopicHandler(context context.Context, clusterName string, spec *eventingv1alpha1.AzureEventGridTopicSpec, logger logr.Logger) (*AzureEventGridTopicHandler, error) {
	client, err := publisher.NewClientWithSharedKeyCredential(spec.EndPoint, azcore.NewKeyCredential(spec.Key), nil)
	if err != nil {
		return nil, err
	}

	logger.Info("Create new azure event grid handler")
	return &AzureEventGridTopicHandler{
		Context:      context,
		Client:       client,
		Endpoint:     spec.EndPoint,
		Key:          spec.Key,
		ClusterName:  clusterName,
		logger:       logger,
		activeStatus: metav1.ConditionTrue,
	}, nil
}

func (a *AzureEventGridTopicHandler) CloseHandler() {

}

func (a *AzureEventGridTopicHandler) SetActiveStatus(status metav1.ConditionStatus) {
	a.activeStatus = status
}

func (a *AzureEventGridTopicHandler) GetActiveStatus() metav1.ConditionStatus {
	return a.activeStatus
}

func (a *AzureEventGridTopicHandler) EmitEvent(eventData eventdata.EventData, failureFunc func(eventData eventdata.EventData, err error)) {
	type emitData struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}

	source := fmt.Sprintf("/%s/%s/keda", a.ClusterName, kedaNamespace)
	subject := fmt.Sprintf("/%s/%s/%s/%s", a.ClusterName, eventData.Namespace, eventData.ObjectType, eventData.ObjectName)

	opt := &messaging.CloudEventOptions{
		Subject:         &subject,
		DataContentType: to.Ptr("application/json"),
		Time:            &eventData.Time,
	}

	event, err := messaging.NewCloudEvent(source, eventData.EventType, emitData{Reason: eventData.Reason, Message: eventData.Message}, opt)

	if err != nil {
		a.logger.Error(err, "EmitEvent error %s")
		return
	}

	eventsToSend := []messaging.CloudEvent{
		event,
	}

	_, err = a.Client.PublishCloudEvents(a.Context, eventsToSend, &publisher.PublishCloudEventsOptions{})

	if err != nil {
		a.logger.Error(err, "Failed to Publish Event to Azure Event Grid ")
		failureFunc(eventData, err)
		return
	}

	a.logger.Info("Publish Event to Azure Event Grid Successfully")
}
