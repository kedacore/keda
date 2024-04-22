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
// CloudEventHTTPHandler focuses on emitting the CloudEventSource to CloudEvent
// HTTP URI. URI can be defined in CloudEventSourceSpec.
// ************************************************************************** \\

package eventemitter

import (
	"context"
	"fmt"
	"net/url"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
)

type CloudEventHTTPHandler struct {
	ctx          context.Context
	logger       logr.Logger
	endpoint     string
	client       cloudevents.Client
	clusterName  string
	activeStatus metav1.ConditionStatus
}

func NewCloudEventHTTPHandler(context context.Context, clusterName string, uri string, logger logr.Logger) (*CloudEventHTTPHandler, error) {
	if uri == "" {
		return nil, fmt.Errorf("uri cannot be empty")
	}

	if _, err := url.ParseRequestURI(uri); err != nil {
		return nil, err
	}

	client, err := cloudevents.NewClientHTTP()
	ctx := cloudevents.ContextWithTarget(context, uri)
	if err != nil {
		return nil, err
	}

	logger.Info("Create new cloudevents http handler with endPoint: " + uri)
	return &CloudEventHTTPHandler{
		client:       client,
		endpoint:     uri,
		clusterName:  clusterName,
		activeStatus: metav1.ConditionTrue,
		ctx:          ctx,
		logger:       logger,
	}, nil
}

func (c *CloudEventHTTPHandler) SetActiveStatus(status metav1.ConditionStatus) {
	c.activeStatus = status
}

func (c *CloudEventHTTPHandler) GetActiveStatus() metav1.ConditionStatus {
	return c.activeStatus
}

func (c *CloudEventHTTPHandler) CloseHandler() {
	c.logger.V(1).Info("Closing CloudEvent HTTP handler")
}

func (c *CloudEventHTTPHandler) EmitEvent(eventData eventdata.EventData, failureFunc func(eventData eventdata.EventData, err error)) {
	source := generateCloudEventSource(c.clusterName)
	subject := generateCloudEventSubjectFromEventData(c.clusterName, eventData)

	event := cloudevents.NewEvent()
	event.SetSource(source)
	event.SetSubject(subject)
	event.SetType(string(eventData.CloudEventType))

	if err := event.SetData(cloudevents.ApplicationJSON, EmitData{Reason: eventData.Reason, Message: eventData.Message}); err != nil {
		c.logger.Error(err, "Failed to set data to CloudEvents receiver")
		return
	}

	err := c.client.Send(c.ctx, event)
	if protocol.IsNACK(err) || protocol.IsUndelivered(err) {
		c.logger.Error(err, "Failed to send event to CloudEvents receiver")
		failureFunc(eventData, err)
		return
	}

	c.logger.V(1).Info("Successfully published event to CloudEvents receiver")
}
