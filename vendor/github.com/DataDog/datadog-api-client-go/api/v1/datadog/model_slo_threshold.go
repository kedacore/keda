// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOThreshold SLO thresholds (target and optionally warning) for a single time window.
type SLOThreshold struct {
	// The target value for the service level indicator within the corresponding
	// timeframe.
	Target float64 `json:"target"`
	// A string representation of the target that indicates its precision.
	// It uses trailing zeros to show significant decimal places (for example `98.00`).
	//
	// Always included in service level objective responses. Ignored in
	// create/update requests.
	TargetDisplay *string `json:"target_display,omitempty"`
	// The SLO time window options.
	Timeframe SLOTimeframe `json:"timeframe"`
	// The warning value for the service level objective.
	Warning *float64 `json:"warning,omitempty"`
	// A string representation of the warning target (see the description of
	// the `target_display` field for details).
	//
	// Included in service level objective responses if a warning target exists.
	// Ignored in create/update requests.
	WarningDisplay *string `json:"warning_display,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOThreshold instantiates a new SLOThreshold object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOThreshold(target float64, timeframe SLOTimeframe) *SLOThreshold {
	this := SLOThreshold{}
	this.Target = target
	this.Timeframe = timeframe
	return &this
}

// NewSLOThresholdWithDefaults instantiates a new SLOThreshold object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOThresholdWithDefaults() *SLOThreshold {
	this := SLOThreshold{}
	return &this
}

// GetTarget returns the Target field value.
func (o *SLOThreshold) GetTarget() float64 {
	if o == nil {
		var ret float64
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *SLOThreshold) GetTargetOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *SLOThreshold) SetTarget(v float64) {
	o.Target = v
}

// GetTargetDisplay returns the TargetDisplay field value if set, zero value otherwise.
func (o *SLOThreshold) GetTargetDisplay() string {
	if o == nil || o.TargetDisplay == nil {
		var ret string
		return ret
	}
	return *o.TargetDisplay
}

// GetTargetDisplayOk returns a tuple with the TargetDisplay field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOThreshold) GetTargetDisplayOk() (*string, bool) {
	if o == nil || o.TargetDisplay == nil {
		return nil, false
	}
	return o.TargetDisplay, true
}

// HasTargetDisplay returns a boolean if a field has been set.
func (o *SLOThreshold) HasTargetDisplay() bool {
	if o != nil && o.TargetDisplay != nil {
		return true
	}

	return false
}

// SetTargetDisplay gets a reference to the given string and assigns it to the TargetDisplay field.
func (o *SLOThreshold) SetTargetDisplay(v string) {
	o.TargetDisplay = &v
}

// GetTimeframe returns the Timeframe field value.
func (o *SLOThreshold) GetTimeframe() SLOTimeframe {
	if o == nil {
		var ret SLOTimeframe
		return ret
	}
	return o.Timeframe
}

// GetTimeframeOk returns a tuple with the Timeframe field value
// and a boolean to check if the value has been set.
func (o *SLOThreshold) GetTimeframeOk() (*SLOTimeframe, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Timeframe, true
}

// SetTimeframe sets field value.
func (o *SLOThreshold) SetTimeframe(v SLOTimeframe) {
	o.Timeframe = v
}

// GetWarning returns the Warning field value if set, zero value otherwise.
func (o *SLOThreshold) GetWarning() float64 {
	if o == nil || o.Warning == nil {
		var ret float64
		return ret
	}
	return *o.Warning
}

// GetWarningOk returns a tuple with the Warning field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOThreshold) GetWarningOk() (*float64, bool) {
	if o == nil || o.Warning == nil {
		return nil, false
	}
	return o.Warning, true
}

// HasWarning returns a boolean if a field has been set.
func (o *SLOThreshold) HasWarning() bool {
	if o != nil && o.Warning != nil {
		return true
	}

	return false
}

// SetWarning gets a reference to the given float64 and assigns it to the Warning field.
func (o *SLOThreshold) SetWarning(v float64) {
	o.Warning = &v
}

// GetWarningDisplay returns the WarningDisplay field value if set, zero value otherwise.
func (o *SLOThreshold) GetWarningDisplay() string {
	if o == nil || o.WarningDisplay == nil {
		var ret string
		return ret
	}
	return *o.WarningDisplay
}

// GetWarningDisplayOk returns a tuple with the WarningDisplay field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOThreshold) GetWarningDisplayOk() (*string, bool) {
	if o == nil || o.WarningDisplay == nil {
		return nil, false
	}
	return o.WarningDisplay, true
}

// HasWarningDisplay returns a boolean if a field has been set.
func (o *SLOThreshold) HasWarningDisplay() bool {
	if o != nil && o.WarningDisplay != nil {
		return true
	}

	return false
}

// SetWarningDisplay gets a reference to the given string and assigns it to the WarningDisplay field.
func (o *SLOThreshold) SetWarningDisplay(v string) {
	o.WarningDisplay = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOThreshold) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["target"] = o.Target
	if o.TargetDisplay != nil {
		toSerialize["target_display"] = o.TargetDisplay
	}
	toSerialize["timeframe"] = o.Timeframe
	if o.Warning != nil {
		toSerialize["warning"] = o.Warning
	}
	if o.WarningDisplay != nil {
		toSerialize["warning_display"] = o.WarningDisplay
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOThreshold) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Target    *float64      `json:"target"`
		Timeframe *SLOTimeframe `json:"timeframe"`
	}{}
	all := struct {
		Target         float64      `json:"target"`
		TargetDisplay  *string      `json:"target_display,omitempty"`
		Timeframe      SLOTimeframe `json:"timeframe"`
		Warning        *float64     `json:"warning,omitempty"`
		WarningDisplay *string      `json:"warning_display,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Target == nil {
		return fmt.Errorf("Required field target missing")
	}
	if required.Timeframe == nil {
		return fmt.Errorf("Required field timeframe missing")
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
	if v := all.Timeframe; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Target = all.Target
	o.TargetDisplay = all.TargetDisplay
	o.Timeframe = all.Timeframe
	o.Warning = all.Warning
	o.WarningDisplay = all.WarningDisplay
	return nil
}
