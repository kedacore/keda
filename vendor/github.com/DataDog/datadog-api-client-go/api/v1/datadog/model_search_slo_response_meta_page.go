// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SearchSLOResponseMetaPage Pagination metadata returned by the API.
type SearchSLOResponseMetaPage struct {
	// The first number.
	FirstNumber *int64 `json:"first_number,omitempty"`
	// The last number.
	LastNumber *int64 `json:"last_number,omitempty"`
	// The next number.
	NextNumber *int64 `json:"next_number,omitempty"`
	// The page number.
	Number *int64 `json:"number,omitempty"`
	// The previous page number.
	PrevNumber *int64 `json:"prev_number,omitempty"`
	// The size of the response.
	Size *int64 `json:"size,omitempty"`
	// The total number of SLOs in the response.
	Total *int64 `json:"total,omitempty"`
	// Type of pagination.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOResponseMetaPage instantiates a new SearchSLOResponseMetaPage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOResponseMetaPage() *SearchSLOResponseMetaPage {
	this := SearchSLOResponseMetaPage{}
	return &this
}

// NewSearchSLOResponseMetaPageWithDefaults instantiates a new SearchSLOResponseMetaPage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOResponseMetaPageWithDefaults() *SearchSLOResponseMetaPage {
	this := SearchSLOResponseMetaPage{}
	return &this
}

// GetFirstNumber returns the FirstNumber field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetFirstNumber() int64 {
	if o == nil || o.FirstNumber == nil {
		var ret int64
		return ret
	}
	return *o.FirstNumber
}

// GetFirstNumberOk returns a tuple with the FirstNumber field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetFirstNumberOk() (*int64, bool) {
	if o == nil || o.FirstNumber == nil {
		return nil, false
	}
	return o.FirstNumber, true
}

// HasFirstNumber returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasFirstNumber() bool {
	if o != nil && o.FirstNumber != nil {
		return true
	}

	return false
}

// SetFirstNumber gets a reference to the given int64 and assigns it to the FirstNumber field.
func (o *SearchSLOResponseMetaPage) SetFirstNumber(v int64) {
	o.FirstNumber = &v
}

// GetLastNumber returns the LastNumber field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetLastNumber() int64 {
	if o == nil || o.LastNumber == nil {
		var ret int64
		return ret
	}
	return *o.LastNumber
}

// GetLastNumberOk returns a tuple with the LastNumber field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetLastNumberOk() (*int64, bool) {
	if o == nil || o.LastNumber == nil {
		return nil, false
	}
	return o.LastNumber, true
}

// HasLastNumber returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasLastNumber() bool {
	if o != nil && o.LastNumber != nil {
		return true
	}

	return false
}

// SetLastNumber gets a reference to the given int64 and assigns it to the LastNumber field.
func (o *SearchSLOResponseMetaPage) SetLastNumber(v int64) {
	o.LastNumber = &v
}

// GetNextNumber returns the NextNumber field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetNextNumber() int64 {
	if o == nil || o.NextNumber == nil {
		var ret int64
		return ret
	}
	return *o.NextNumber
}

// GetNextNumberOk returns a tuple with the NextNumber field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetNextNumberOk() (*int64, bool) {
	if o == nil || o.NextNumber == nil {
		return nil, false
	}
	return o.NextNumber, true
}

// HasNextNumber returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasNextNumber() bool {
	if o != nil && o.NextNumber != nil {
		return true
	}

	return false
}

// SetNextNumber gets a reference to the given int64 and assigns it to the NextNumber field.
func (o *SearchSLOResponseMetaPage) SetNextNumber(v int64) {
	o.NextNumber = &v
}

// GetNumber returns the Number field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetNumber() int64 {
	if o == nil || o.Number == nil {
		var ret int64
		return ret
	}
	return *o.Number
}

// GetNumberOk returns a tuple with the Number field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetNumberOk() (*int64, bool) {
	if o == nil || o.Number == nil {
		return nil, false
	}
	return o.Number, true
}

// HasNumber returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasNumber() bool {
	if o != nil && o.Number != nil {
		return true
	}

	return false
}

// SetNumber gets a reference to the given int64 and assigns it to the Number field.
func (o *SearchSLOResponseMetaPage) SetNumber(v int64) {
	o.Number = &v
}

// GetPrevNumber returns the PrevNumber field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetPrevNumber() int64 {
	if o == nil || o.PrevNumber == nil {
		var ret int64
		return ret
	}
	return *o.PrevNumber
}

// GetPrevNumberOk returns a tuple with the PrevNumber field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetPrevNumberOk() (*int64, bool) {
	if o == nil || o.PrevNumber == nil {
		return nil, false
	}
	return o.PrevNumber, true
}

// HasPrevNumber returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasPrevNumber() bool {
	if o != nil && o.PrevNumber != nil {
		return true
	}

	return false
}

// SetPrevNumber gets a reference to the given int64 and assigns it to the PrevNumber field.
func (o *SearchSLOResponseMetaPage) SetPrevNumber(v int64) {
	o.PrevNumber = &v
}

// GetSize returns the Size field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetSize() int64 {
	if o == nil || o.Size == nil {
		var ret int64
		return ret
	}
	return *o.Size
}

// GetSizeOk returns a tuple with the Size field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetSizeOk() (*int64, bool) {
	if o == nil || o.Size == nil {
		return nil, false
	}
	return o.Size, true
}

// HasSize returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasSize() bool {
	if o != nil && o.Size != nil {
		return true
	}

	return false
}

// SetSize gets a reference to the given int64 and assigns it to the Size field.
func (o *SearchSLOResponseMetaPage) SetSize(v int64) {
	o.Size = &v
}

// GetTotal returns the Total field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetTotal() int64 {
	if o == nil || o.Total == nil {
		var ret int64
		return ret
	}
	return *o.Total
}

// GetTotalOk returns a tuple with the Total field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetTotalOk() (*int64, bool) {
	if o == nil || o.Total == nil {
		return nil, false
	}
	return o.Total, true
}

// HasTotal returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasTotal() bool {
	if o != nil && o.Total != nil {
		return true
	}

	return false
}

// SetTotal gets a reference to the given int64 and assigns it to the Total field.
func (o *SearchSLOResponseMetaPage) SetTotal(v int64) {
	o.Total = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SearchSLOResponseMetaPage) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseMetaPage) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SearchSLOResponseMetaPage) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *SearchSLOResponseMetaPage) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOResponseMetaPage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.FirstNumber != nil {
		toSerialize["first_number"] = o.FirstNumber
	}
	if o.LastNumber != nil {
		toSerialize["last_number"] = o.LastNumber
	}
	if o.NextNumber != nil {
		toSerialize["next_number"] = o.NextNumber
	}
	if o.Number != nil {
		toSerialize["number"] = o.Number
	}
	if o.PrevNumber != nil {
		toSerialize["prev_number"] = o.PrevNumber
	}
	if o.Size != nil {
		toSerialize["size"] = o.Size
	}
	if o.Total != nil {
		toSerialize["total"] = o.Total
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOResponseMetaPage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		FirstNumber *int64  `json:"first_number,omitempty"`
		LastNumber  *int64  `json:"last_number,omitempty"`
		NextNumber  *int64  `json:"next_number,omitempty"`
		Number      *int64  `json:"number,omitempty"`
		PrevNumber  *int64  `json:"prev_number,omitempty"`
		Size        *int64  `json:"size,omitempty"`
		Total       *int64  `json:"total,omitempty"`
		Type        *string `json:"type,omitempty"`
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
	o.FirstNumber = all.FirstNumber
	o.LastNumber = all.LastNumber
	o.NextNumber = all.NextNumber
	o.Number = all.Number
	o.PrevNumber = all.PrevNumber
	o.Size = all.Size
	o.Total = all.Total
	o.Type = all.Type
	return nil
}
