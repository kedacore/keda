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
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type AzureEventGridHandler struct {
	Endpoint         string
	Key              string
	TopicName        string
	SubscriptionName string
	Client           *azeventgrid.Client
}

func NewAzureEventGridHandler(spec kedav1alpha1.AzureEventGridSpec) (*AzureEventGridHandler, error) {
	client, err := azeventgrid.NewClientWithSharedKeyCredential(spec.EndPoint, spec.Key, nil)

	if err != nil {
		return nil, err
	}
	fmt.Print("new azure event grid handler....")
	return &AzureEventGridHandler{Client: client, Endpoint: spec.EndPoint, Key: spec.Key, TopicName: spec.TopicName, SubscriptionName: spec.SubscriptionName}, nil
}

func (a *AzureEventGridHandler) CloseHandler() {

}

func (a *AzureEventGridHandler) EmitEvent(eventData EventData, failureFunc func(eventData EventData, err error)) error {

	type SampleData struct {
		Name string `json:"name"`
	}

	event, err := messaging.NewCloudEvent("testsource", "testeventType", SampleData{Name: "hello"}, nil)

	if err != nil {
		fmt.Printf("EmitEvent error %s", err.Error())
		return err
	}

	eventsToSend := []messaging.CloudEvent{
		event,
	}

	// NOTE: we're sending a single event as an example. For better efficiency it's best if you send
	// multiple events at a time.
	response, err := a.Client.PublishCloudEvents(context.TODO(), a.TopicName, eventsToSend, nil)

	if err != nil {
		fmt.Printf("Publish failed %s", err.Error())
		failureFunc(eventData, err)
	}
	fmt.Printf("Publish successfully %s", response)
	return nil
}
