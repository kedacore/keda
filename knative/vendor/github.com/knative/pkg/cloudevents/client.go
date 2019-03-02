/*
Copyright 2019 The Knative Authors

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

package cloudevents

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client wraps Builder, and is intended to be configured for a single event
// type and target
type Client struct {
	builder Builder
	Target  string
}

// NewClient returns a CloudEvent Client used to send CloudEvents. It is
// intended that a user would create a new client for each tuple of eventType
// and target. This is an optional helper method to avoid the tricky creation
// of the embedded Builder struct.
func NewClient(target string, builder Builder) *Client {
	c := &Client{
		builder: builder,
		Target:  target,
	}
	return c
}

// Send creates a request based on the client's settings and sends the data
// struct to the target set for this client. It returns error if there was an
// issue sending the event, otherwise nil means the event was accepted.
func (c *Client) Send(data interface{}, overrides ...SendContext) error {
	req, err := c.builder.Build(c.Target, data, overrides...)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if accepted(resp) {
		return nil
	}
	return fmt.Errorf("error sending cloudevent: %s", status(resp))
}

// accepted is a helper method to understand if the response from the target
// accepted the CloudEvent.
func accepted(resp *http.Response) bool {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}
	return false
}

// status is a helper method to read the response of the target.
func status(resp *http.Response) string {
	status := resp.Status
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Status[%s] error reading response body: %v", status, err)
	}
	return fmt.Sprintf("Status[%s] %s", status, body)
}
