// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MetricsQueryResponse Response Object that includes your query and the list of metrics retrieved.
type MetricsQueryResponse struct {
	// Message indicating the errors if status is not `ok`.
	Error *string `json:"error,omitempty"`
	// Start of requested time window, milliseconds since Unix epoch.
	FromDate *int64 `json:"from_date,omitempty"`
	// List of tag keys on which to group.
	GroupBy []string `json:"group_by,omitempty"`
	// Message indicating `success` if status is `ok`.
	Message *string `json:"message,omitempty"`
	// Query string
	Query *string `json:"query,omitempty"`
	// Type of response.
	ResType *string `json:"res_type,omitempty"`
	// List of timeseries queried.
	Series []MetricsQueryMetadata `json:"series,omitempty"`
	// Status of the query.
	Status *string `json:"status,omitempty"`
	// End of requested time window, milliseconds since Unix epoch.
	ToDate *int64 `json:"to_date,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMetricsQueryResponse instantiates a new MetricsQueryResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricsQueryResponse() *MetricsQueryResponse {
	this := MetricsQueryResponse{}
	return &this
}

// NewMetricsQueryResponseWithDefaults instantiates a new MetricsQueryResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricsQueryResponseWithDefaults() *MetricsQueryResponse {
	this := MetricsQueryResponse{}
	return &this
}

// GetError returns the Error field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetError() string {
	if o == nil || o.Error == nil {
		var ret string
		return ret
	}
	return *o.Error
}

// GetErrorOk returns a tuple with the Error field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetErrorOk() (*string, bool) {
	if o == nil || o.Error == nil {
		return nil, false
	}
	return o.Error, true
}

// HasError returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasError() bool {
	if o != nil && o.Error != nil {
		return true
	}

	return false
}

// SetError gets a reference to the given string and assigns it to the Error field.
func (o *MetricsQueryResponse) SetError(v string) {
	o.Error = &v
}

// GetFromDate returns the FromDate field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetFromDate() int64 {
	if o == nil || o.FromDate == nil {
		var ret int64
		return ret
	}
	return *o.FromDate
}

// GetFromDateOk returns a tuple with the FromDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetFromDateOk() (*int64, bool) {
	if o == nil || o.FromDate == nil {
		return nil, false
	}
	return o.FromDate, true
}

// HasFromDate returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasFromDate() bool {
	if o != nil && o.FromDate != nil {
		return true
	}

	return false
}

// SetFromDate gets a reference to the given int64 and assigns it to the FromDate field.
func (o *MetricsQueryResponse) SetFromDate(v int64) {
	o.FromDate = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetGroupBy() []string {
	if o == nil || o.GroupBy == nil {
		var ret []string
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetGroupByOk() (*[]string, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasGroupBy() bool {
	if o != nil && o.GroupBy != nil {
		return true
	}

	return false
}

// SetGroupBy gets a reference to the given []string and assigns it to the GroupBy field.
func (o *MetricsQueryResponse) SetGroupBy(v []string) {
	o.GroupBy = v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *MetricsQueryResponse) SetMessage(v string) {
	o.Message = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *MetricsQueryResponse) SetQuery(v string) {
	o.Query = &v
}

// GetResType returns the ResType field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetResType() string {
	if o == nil || o.ResType == nil {
		var ret string
		return ret
	}
	return *o.ResType
}

// GetResTypeOk returns a tuple with the ResType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetResTypeOk() (*string, bool) {
	if o == nil || o.ResType == nil {
		return nil, false
	}
	return o.ResType, true
}

// HasResType returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasResType() bool {
	if o != nil && o.ResType != nil {
		return true
	}

	return false
}

// SetResType gets a reference to the given string and assigns it to the ResType field.
func (o *MetricsQueryResponse) SetResType(v string) {
	o.ResType = &v
}

// GetSeries returns the Series field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetSeries() []MetricsQueryMetadata {
	if o == nil || o.Series == nil {
		var ret []MetricsQueryMetadata
		return ret
	}
	return o.Series
}

// GetSeriesOk returns a tuple with the Series field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetSeriesOk() (*[]MetricsQueryMetadata, bool) {
	if o == nil || o.Series == nil {
		return nil, false
	}
	return &o.Series, true
}

// HasSeries returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasSeries() bool {
	if o != nil && o.Series != nil {
		return true
	}

	return false
}

// SetSeries gets a reference to the given []MetricsQueryMetadata and assigns it to the Series field.
func (o *MetricsQueryResponse) SetSeries(v []MetricsQueryMetadata) {
	o.Series = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *MetricsQueryResponse) SetStatus(v string) {
	o.Status = &v
}

// GetToDate returns the ToDate field value if set, zero value otherwise.
func (o *MetricsQueryResponse) GetToDate() int64 {
	if o == nil || o.ToDate == nil {
		var ret int64
		return ret
	}
	return *o.ToDate
}

// GetToDateOk returns a tuple with the ToDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryResponse) GetToDateOk() (*int64, bool) {
	if o == nil || o.ToDate == nil {
		return nil, false
	}
	return o.ToDate, true
}

// HasToDate returns a boolean if a field has been set.
func (o *MetricsQueryResponse) HasToDate() bool {
	if o != nil && o.ToDate != nil {
		return true
	}

	return false
}

// SetToDate gets a reference to the given int64 and assigns it to the ToDate field.
func (o *MetricsQueryResponse) SetToDate(v int64) {
	o.ToDate = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricsQueryResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Error != nil {
		toSerialize["error"] = o.Error
	}
	if o.FromDate != nil {
		toSerialize["from_date"] = o.FromDate
	}
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.ResType != nil {
		toSerialize["res_type"] = o.ResType
	}
	if o.Series != nil {
		toSerialize["series"] = o.Series
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.ToDate != nil {
		toSerialize["to_date"] = o.ToDate
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricsQueryResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Error    *string                `json:"error,omitempty"`
		FromDate *int64                 `json:"from_date,omitempty"`
		GroupBy  []string               `json:"group_by,omitempty"`
		Message  *string                `json:"message,omitempty"`
		Query    *string                `json:"query,omitempty"`
		ResType  *string                `json:"res_type,omitempty"`
		Series   []MetricsQueryMetadata `json:"series,omitempty"`
		Status   *string                `json:"status,omitempty"`
		ToDate   *int64                 `json:"to_date,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Error = all.Error
	o.FromDate = all.FromDate
	o.GroupBy = all.GroupBy
	o.Message = all.Message
	o.Query = all.Query
	o.ResType = all.ResType
	o.Series = all.Series
	o.Status = all.Status
	o.ToDate = all.ToDate
	return nil
}
