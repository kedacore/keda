// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsGrokParserRules Set of rules for the grok parser.
type LogsGrokParserRules struct {
	// List of match rules for the grok parser, separated by a new line.
	MatchRules string `json:"match_rules"`
	// List of support rules for the grok parser, separated by a new line.
	SupportRules *string `json:"support_rules,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsGrokParserRules instantiates a new LogsGrokParserRules object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsGrokParserRules(matchRules string) *LogsGrokParserRules {
	this := LogsGrokParserRules{}
	this.MatchRules = matchRules
	var supportRules string = ""
	this.SupportRules = &supportRules
	return &this
}

// NewLogsGrokParserRulesWithDefaults instantiates a new LogsGrokParserRules object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsGrokParserRulesWithDefaults() *LogsGrokParserRules {
	this := LogsGrokParserRules{}
	var supportRules string = ""
	this.SupportRules = &supportRules
	return &this
}

// GetMatchRules returns the MatchRules field value.
func (o *LogsGrokParserRules) GetMatchRules() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.MatchRules
}

// GetMatchRulesOk returns a tuple with the MatchRules field value
// and a boolean to check if the value has been set.
func (o *LogsGrokParserRules) GetMatchRulesOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.MatchRules, true
}

// SetMatchRules sets field value.
func (o *LogsGrokParserRules) SetMatchRules(v string) {
	o.MatchRules = v
}

// GetSupportRules returns the SupportRules field value if set, zero value otherwise.
func (o *LogsGrokParserRules) GetSupportRules() string {
	if o == nil || o.SupportRules == nil {
		var ret string
		return ret
	}
	return *o.SupportRules
}

// GetSupportRulesOk returns a tuple with the SupportRules field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGrokParserRules) GetSupportRulesOk() (*string, bool) {
	if o == nil || o.SupportRules == nil {
		return nil, false
	}
	return o.SupportRules, true
}

// HasSupportRules returns a boolean if a field has been set.
func (o *LogsGrokParserRules) HasSupportRules() bool {
	if o != nil && o.SupportRules != nil {
		return true
	}

	return false
}

// SetSupportRules gets a reference to the given string and assigns it to the SupportRules field.
func (o *LogsGrokParserRules) SetSupportRules(v string) {
	o.SupportRules = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsGrokParserRules) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["match_rules"] = o.MatchRules
	if o.SupportRules != nil {
		toSerialize["support_rules"] = o.SupportRules
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsGrokParserRules) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		MatchRules *string `json:"match_rules"`
	}{}
	all := struct {
		MatchRules   string  `json:"match_rules"`
		SupportRules *string `json:"support_rules,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.MatchRules == nil {
		return fmt.Errorf("Required field match_rules missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.MatchRules = all.MatchRules
	o.SupportRules = all.SupportRules
	return nil
}
