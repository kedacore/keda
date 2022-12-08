// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsListRequest Object to send with the request to retrieve a list of logs from your Organization.
type LogsListRequest struct {
	// The log index on which the request is performed. For multi-index organizations,
	// the default is all live indexes. Historical indexes of rehydrated logs must be specified.
	Index *string `json:"index,omitempty"`
	// Number of logs return in the response.
	Limit *int32 `json:"limit,omitempty"`
	// The search query - following the log search syntax.
	Query *string `json:"query,omitempty"`
	// Time-ascending `asc` or time-descending `desc` results.
	Sort *LogsSort `json:"sort,omitempty"`
	// Hash identifier of the first log to return in the list, available in a log `id` attribute.
	// This parameter is used for the pagination feature.
	//
	// **Note**: This parameter is ignored if the corresponding log
	// is out of the scope of the specified time window.
	StartAt *string `json:"startAt,omitempty"`
	// Timeframe to retrieve the log from.
	Time LogsListRequestTime `json:"time"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsListRequest instantiates a new LogsListRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsListRequest(time LogsListRequestTime) *LogsListRequest {
	this := LogsListRequest{}
	this.Time = time
	return &this
}

// NewLogsListRequestWithDefaults instantiates a new LogsListRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsListRequestWithDefaults() *LogsListRequest {
	this := LogsListRequest{}
	return &this
}

// GetIndex returns the Index field value if set, zero value otherwise.
func (o *LogsListRequest) GetIndex() string {
	if o == nil || o.Index == nil {
		var ret string
		return ret
	}
	return *o.Index
}

// GetIndexOk returns a tuple with the Index field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequest) GetIndexOk() (*string, bool) {
	if o == nil || o.Index == nil {
		return nil, false
	}
	return o.Index, true
}

// HasIndex returns a boolean if a field has been set.
func (o *LogsListRequest) HasIndex() bool {
	if o != nil && o.Index != nil {
		return true
	}

	return false
}

// SetIndex gets a reference to the given string and assigns it to the Index field.
func (o *LogsListRequest) SetIndex(v string) {
	o.Index = &v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *LogsListRequest) GetLimit() int32 {
	if o == nil || o.Limit == nil {
		var ret int32
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequest) GetLimitOk() (*int32, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *LogsListRequest) HasLimit() bool {
	if o != nil && o.Limit != nil {
		return true
	}

	return false
}

// SetLimit gets a reference to the given int32 and assigns it to the Limit field.
func (o *LogsListRequest) SetLimit(v int32) {
	o.Limit = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *LogsListRequest) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequest) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *LogsListRequest) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *LogsListRequest) SetQuery(v string) {
	o.Query = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *LogsListRequest) GetSort() LogsSort {
	if o == nil || o.Sort == nil {
		var ret LogsSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequest) GetSortOk() (*LogsSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *LogsListRequest) HasSort() bool {
	if o != nil && o.Sort != nil {
		return true
	}

	return false
}

// SetSort gets a reference to the given LogsSort and assigns it to the Sort field.
func (o *LogsListRequest) SetSort(v LogsSort) {
	o.Sort = &v
}

// GetStartAt returns the StartAt field value if set, zero value otherwise.
func (o *LogsListRequest) GetStartAt() string {
	if o == nil || o.StartAt == nil {
		var ret string
		return ret
	}
	return *o.StartAt
}

// GetStartAtOk returns a tuple with the StartAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequest) GetStartAtOk() (*string, bool) {
	if o == nil || o.StartAt == nil {
		return nil, false
	}
	return o.StartAt, true
}

// HasStartAt returns a boolean if a field has been set.
func (o *LogsListRequest) HasStartAt() bool {
	if o != nil && o.StartAt != nil {
		return true
	}

	return false
}

// SetStartAt gets a reference to the given string and assigns it to the StartAt field.
func (o *LogsListRequest) SetStartAt(v string) {
	o.StartAt = &v
}

// GetTime returns the Time field value.
func (o *LogsListRequest) GetTime() LogsListRequestTime {
	if o == nil {
		var ret LogsListRequestTime
		return ret
	}
	return o.Time
}

// GetTimeOk returns a tuple with the Time field value
// and a boolean to check if the value has been set.
func (o *LogsListRequest) GetTimeOk() (*LogsListRequestTime, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Time, true
}

// SetTime sets field value.
func (o *LogsListRequest) SetTime(v LogsListRequestTime) {
	o.Time = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsListRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Index != nil {
		toSerialize["index"] = o.Index
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}
	if o.StartAt != nil {
		toSerialize["startAt"] = o.StartAt
	}
	toSerialize["time"] = o.Time

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsListRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Time *LogsListRequestTime `json:"time"`
	}{}
	all := struct {
		Index   *string             `json:"index,omitempty"`
		Limit   *int32              `json:"limit,omitempty"`
		Query   *string             `json:"query,omitempty"`
		Sort    *LogsSort           `json:"sort,omitempty"`
		StartAt *string             `json:"startAt,omitempty"`
		Time    LogsListRequestTime `json:"time"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Time == nil {
		return fmt.Errorf("Required field time missing")
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
	if v := all.Sort; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Index = all.Index
	o.Limit = all.Limit
	o.Query = all.Query
	o.Sort = all.Sort
	o.StartAt = all.StartAt
	if all.Time.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Time = all.Time
	return nil
}
