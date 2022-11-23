// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOHistoryMetrics A `metric` based SLO history response.
//
// This is not included in responses for `monitor` based SLOs.
type SLOHistoryMetrics struct {
	// A representation of `metric` based SLO time series for the provided queries.
	// This is the same response type from `batch_query` endpoint.
	Denominator SLOHistoryMetricsSeries `json:"denominator"`
	// The aggregated query interval for the series data. It's implicit based on the query time window.
	Interval int64 `json:"interval"`
	// Optional message if there are specific query issues/warnings.
	Message *string `json:"message,omitempty"`
	// A representation of `metric` based SLO time series for the provided queries.
	// This is the same response type from `batch_query` endpoint.
	Numerator SLOHistoryMetricsSeries `json:"numerator"`
	// The combined numerator and denominator query CSV.
	Query string `json:"query"`
	// The series result type. This mimics `batch_query` response type.
	ResType string `json:"res_type"`
	// The series response version type. This mimics `batch_query` response type.
	RespVersion int64 `json:"resp_version"`
	// An array of query timestamps in EPOCH milliseconds.
	Times []float64 `json:"times"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryMetrics instantiates a new SLOHistoryMetrics object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryMetrics(denominator SLOHistoryMetricsSeries, interval int64, numerator SLOHistoryMetricsSeries, query string, resType string, respVersion int64, times []float64) *SLOHistoryMetrics {
	this := SLOHistoryMetrics{}
	this.Denominator = denominator
	this.Interval = interval
	this.Numerator = numerator
	this.Query = query
	this.ResType = resType
	this.RespVersion = respVersion
	this.Times = times
	return &this
}

// NewSLOHistoryMetricsWithDefaults instantiates a new SLOHistoryMetrics object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryMetricsWithDefaults() *SLOHistoryMetrics {
	this := SLOHistoryMetrics{}
	return &this
}

// GetDenominator returns the Denominator field value.
func (o *SLOHistoryMetrics) GetDenominator() SLOHistoryMetricsSeries {
	if o == nil {
		var ret SLOHistoryMetricsSeries
		return ret
	}
	return o.Denominator
}

// GetDenominatorOk returns a tuple with the Denominator field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetDenominatorOk() (*SLOHistoryMetricsSeries, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Denominator, true
}

// SetDenominator sets field value.
func (o *SLOHistoryMetrics) SetDenominator(v SLOHistoryMetricsSeries) {
	o.Denominator = v
}

// GetInterval returns the Interval field value.
func (o *SLOHistoryMetrics) GetInterval() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetIntervalOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Interval, true
}

// SetInterval sets field value.
func (o *SLOHistoryMetrics) SetInterval(v int64) {
	o.Interval = v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SLOHistoryMetrics) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SLOHistoryMetrics) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *SLOHistoryMetrics) SetMessage(v string) {
	o.Message = &v
}

// GetNumerator returns the Numerator field value.
func (o *SLOHistoryMetrics) GetNumerator() SLOHistoryMetricsSeries {
	if o == nil {
		var ret SLOHistoryMetricsSeries
		return ret
	}
	return o.Numerator
}

// GetNumeratorOk returns a tuple with the Numerator field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetNumeratorOk() (*SLOHistoryMetricsSeries, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Numerator, true
}

// SetNumerator sets field value.
func (o *SLOHistoryMetrics) SetNumerator(v SLOHistoryMetricsSeries) {
	o.Numerator = v
}

// GetQuery returns the Query field value.
func (o *SLOHistoryMetrics) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *SLOHistoryMetrics) SetQuery(v string) {
	o.Query = v
}

// GetResType returns the ResType field value.
func (o *SLOHistoryMetrics) GetResType() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ResType
}

// GetResTypeOk returns a tuple with the ResType field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetResTypeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ResType, true
}

// SetResType sets field value.
func (o *SLOHistoryMetrics) SetResType(v string) {
	o.ResType = v
}

// GetRespVersion returns the RespVersion field value.
func (o *SLOHistoryMetrics) GetRespVersion() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.RespVersion
}

// GetRespVersionOk returns a tuple with the RespVersion field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetRespVersionOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RespVersion, true
}

// SetRespVersion sets field value.
func (o *SLOHistoryMetrics) SetRespVersion(v int64) {
	o.RespVersion = v
}

// GetTimes returns the Times field value.
func (o *SLOHistoryMetrics) GetTimes() []float64 {
	if o == nil {
		var ret []float64
		return ret
	}
	return o.Times
}

// GetTimesOk returns a tuple with the Times field value
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetrics) GetTimesOk() (*[]float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Times, true
}

// SetTimes sets field value.
func (o *SLOHistoryMetrics) SetTimes(v []float64) {
	o.Times = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryMetrics) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["denominator"] = o.Denominator
	toSerialize["interval"] = o.Interval
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	toSerialize["numerator"] = o.Numerator
	toSerialize["query"] = o.Query
	toSerialize["res_type"] = o.ResType
	toSerialize["resp_version"] = o.RespVersion
	toSerialize["times"] = o.Times

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryMetrics) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Denominator *SLOHistoryMetricsSeries `json:"denominator"`
		Interval    *int64                   `json:"interval"`
		Numerator   *SLOHistoryMetricsSeries `json:"numerator"`
		Query       *string                  `json:"query"`
		ResType     *string                  `json:"res_type"`
		RespVersion *int64                   `json:"resp_version"`
		Times       *[]float64               `json:"times"`
	}{}
	all := struct {
		Denominator SLOHistoryMetricsSeries `json:"denominator"`
		Interval    int64                   `json:"interval"`
		Message     *string                 `json:"message,omitempty"`
		Numerator   SLOHistoryMetricsSeries `json:"numerator"`
		Query       string                  `json:"query"`
		ResType     string                  `json:"res_type"`
		RespVersion int64                   `json:"resp_version"`
		Times       []float64               `json:"times"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Denominator == nil {
		return fmt.Errorf("Required field denominator missing")
	}
	if required.Interval == nil {
		return fmt.Errorf("Required field interval missing")
	}
	if required.Numerator == nil {
		return fmt.Errorf("Required field numerator missing")
	}
	if required.Query == nil {
		return fmt.Errorf("Required field query missing")
	}
	if required.ResType == nil {
		return fmt.Errorf("Required field res_type missing")
	}
	if required.RespVersion == nil {
		return fmt.Errorf("Required field resp_version missing")
	}
	if required.Times == nil {
		return fmt.Errorf("Required field times missing")
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
	if all.Denominator.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Denominator = all.Denominator
	o.Interval = all.Interval
	o.Message = all.Message
	if all.Numerator.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Numerator = all.Numerator
	o.Query = all.Query
	o.ResType = all.ResType
	o.RespVersion = all.RespVersion
	o.Times = all.Times
	return nil
}
