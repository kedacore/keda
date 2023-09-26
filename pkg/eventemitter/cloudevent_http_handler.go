/*
Copyright 2023 The KEDA Authors

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
// CloudEventHTTPHandler focus on emitting the CloudEvents to CloudEvent HTTP
// URI. URI can be defined in CloudEventSpec.
// ************************************************************************** \\

package eventemitter

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/go-logr/logr"
)

type CloudEventHTTPHandler struct {
	Endpoint    string
	Client      cloudevents.Client
	ClusterName string
	ctx         context.Context
	logger      logr.Logger
}

func NewCloudEventHTTPHandler(context context.Context, clusterName string, uri string, logger logr.Logger) (*CloudEventHTTPHandler, error) {
	if uri == "" {
		return nil, fmt.Errorf("uri cannot be empty")
	}

	client, err := cloudevents.NewClientHTTP()
	ctx := cloudevents.ContextWithTarget(context, uri)
	if err != nil {
		return nil, err
	}

	logger.Info("Create new cloudevents http handler with endPoint: " + uri)
	return &CloudEventHTTPHandler{
		Client:      client,
		Endpoint:    uri,
		ClusterName: clusterName,
		ctx:         ctx,
		logger:      logger,
	}, nil
}

func (c *CloudEventHTTPHandler) CloseHandler() {

}

func (c *CloudEventHTTPHandler) EmitEvent(eventData EventData, failureFunc func(eventData EventData, err error)) {
	source := "/" + c.ClusterName + "/" + eventData.namespace + "/keda"
	subject := "/" + c.ClusterName + "/" + eventData.namespace + "/workload/" + eventData.objectName

	event := cloudevents.NewEvent()
	event.SetSource(source)
	event.SetSubject(subject)
	event.SetType(CloudEventType)

	if err := event.SetData(cloudevents.ApplicationJSON, EmitData{Reason: eventData.reason, Message: eventData.message}); err != nil {
		c.logger.Error(err, "Failed to set data to cloudevent")
		return
	}

	err := c.Client.Send(c.ctx, event)
	if protocol.IsNACK(err) || protocol.IsUndelivered(err) {
		c.logger.Error(err, "Failed to send event to cloudevent")
		failureFunc(eventData, err)
		return
	}

	c.logger.V(1).Info("Publish Event to CloudEvents receiver Successfully")
}
