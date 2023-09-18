/*
Copyright 2022 The KEDA Authors

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

package eventemitter

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type AzureEventGridHandler struct {
	Endpoint         string
	Key              string
	TopicName        string
	SubscriptionName string
	ClusterName      string
	Client           *azeventgrid.Client
	logger           logr.Logger
}

func NewAzureEventGridHandler(spec kedav1alpha1.AzureEventGridSpec, clusterName string, logger logr.Logger) (*AzureEventGridHandler, error) {
	client, err := azeventgrid.NewClientWithSharedKeyCredential(spec.EndPoint, spec.Key, nil)

	if err != nil {
		return nil, err
	}

	logger.V(1).Info("Create new azure event grid handler")
	return &AzureEventGridHandler{
		Client:           client,
		Endpoint:         spec.EndPoint,
		Key:              spec.Key,
		TopicName:        spec.TopicName,
		SubscriptionName: spec.SubscriptionName,
		ClusterName:      clusterName,
		logger:           logger,
	}, nil
}

func (a *AzureEventGridHandler) CloseHandler() {

}

func (a *AzureEventGridHandler) EmitEvent(eventData EventData, failureFunc func(eventData EventData, err error)) {
	source := "/" + a.ClusterName + "/" + eventData.namespace + "/keda"
	subject := "/" + a.ClusterName + "/" + eventData.namespace + "/workload/" + eventData.objectName
	opt := &messaging.CloudEventOptions{
		Subject:         &subject,
		DataContentType: to.Ptr("application/json"),
		Time:            &eventData.time,
	}

	event, err := messaging.NewCloudEvent(source, eventData.eventtype, EmitData{Reason: eventData.reason, Message: eventData.message}, opt)

	if err != nil {
		a.logger.Error(err, "EmitEvent error %s")
		return
	}

	eventsToSend := []messaging.CloudEvent{
		event,
	}

	_, err = a.Client.PublishCloudEvents(context.TODO(), a.TopicName, eventsToSend, nil)

	if err != nil {
		a.logger.Error(err, "Failed to Publish Event to Azure Event Grid ")
		failureFunc(eventData, err)
		return
	}

	a.logger.Info("Publish Event to Azure Event Grid Successfully")
}
