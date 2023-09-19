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

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type CloudEventHttpHandler struct {
	Endpoint    string
	Client      cloudevents.Client
	ClusterName string
	ctx         context.Context
	logger      logr.Logger
}

func NewCloudEventHttpHandler(context context.Context, spec kedav1alpha1.CloudEventHttpSpec, clusterName string, logger logr.Logger) (*CloudEventHttpHandler, error) {
	client, err := cloudevents.NewClientHTTP()
	ctx := cloudevents.ContextWithTarget(context, spec.EndPoint)
	if err != nil {
		return nil, err
	}

	logger.V(1).Info("Create new cloudevents http handler")
	return &CloudEventHttpHandler{
		Client:      client,
		Endpoint:    spec.EndPoint,
		ClusterName: clusterName,
		ctx:         ctx,
		logger:      logger,
	}, nil
}

func (c *CloudEventHttpHandler) CloseHandler() {

}

func (c *CloudEventHttpHandler) EmitEvent(eventData EventData, failureFunc func(eventData EventData, err error)) {

	source := "/" + c.ClusterName + "/" + eventData.namespace + "/keda"
	subject := "/" + c.ClusterName + "/" + eventData.namespace + "/workload/" + eventData.objectName

	event := cloudevents.NewEvent()
	event.SetSource(source)
	event.SetType(subject)
	event.SetData(cloudevents.ApplicationJSON, EmitData{Reason: eventData.reason, Message: eventData.message})

	if err := c.Client.Send(c.ctx, event); protocol.IsUndelivered(err) {
		c.logger.Error(err, "Failed to send event to cloudevent")
		failureFunc(eventData, err)
		return
	}

	c.logger.Info("Publish Event to CloudEvents receiver Successfully")
}
