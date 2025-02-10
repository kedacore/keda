// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signalfx/signalfx-go/idtool"
)

type MetadataMessage struct {
	BaseJSONChannelMessage
	TSID       idtool.ID          `json:"tsId"`
	Properties MetadataProperties `json:"properties"`
}

type MetadataProperties struct {
	Metric            string `json:"sf_metric"`
	OriginatingMetric string `json:"sf_originatingMetric"`
	ResolutionMS      int    `json:"sf_resolutionMs"`
	CreatedOnMS       int    `json:"sf_createdOnMs"`
	// Additional SignalFx-generated properties about this time series.  Many
	// of these are exposed directly in fields on this struct.
	InternalProperties map[string]interface{} `json:"-"`
	// Custom properties applied to the timeseries through various means,
	// including dimensions, properties on matching dimensions, etc.
	CustomProperties map[string]string `json:"-"`
}

func (mp *MetadataProperties) UnmarshalJSON(b []byte) error {
	// Deserialize it at first to get all the well-known fields put in place so
	// we don't have to manually assign them below.
	type Alias MetadataProperties
	if err := json.Unmarshal(b, (*Alias)(mp)); err != nil {
		return err
	}

	// Deserialize it again to a generic map so we can get at all the fields.
	var propMap map[string]interface{}
	if err := json.Unmarshal(b, &propMap); err != nil {
		return err
	}

	mp.InternalProperties = make(map[string]interface{})
	mp.CustomProperties = make(map[string]string)
	for k, v := range propMap {
		if strings.HasPrefix(k, "sf_") {
			mp.InternalProperties[k] = v
		} else {
			mp.CustomProperties[k] = fmt.Sprintf("%v", v)
		}
	}
	return nil
}

func (mp *MetadataProperties) MarshalJSON() ([]byte, error) {
	type Alias MetadataProperties
	intermediate, err := json.Marshal((*Alias)(mp))
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	err = json.Unmarshal(intermediate, &out)
	if err != nil {
		return nil, err
	}

	for k, v := range mp.InternalProperties {
		out[k] = v
	}
	for k, v := range mp.CustomProperties {
		out[k] = v
	}

	return json.Marshal(out)
}
