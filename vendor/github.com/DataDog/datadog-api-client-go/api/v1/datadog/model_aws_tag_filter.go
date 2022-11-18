// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSTagFilter A tag filter.
type AWSTagFilter struct {
	// The namespace associated with the tag filter entry.
	Namespace *AWSNamespace `json:"namespace,omitempty"`
	// The tag filter string.
	TagFilterStr *string `json:"tag_filter_str,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSTagFilter instantiates a new AWSTagFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSTagFilter() *AWSTagFilter {
	this := AWSTagFilter{}
	return &this
}

// NewAWSTagFilterWithDefaults instantiates a new AWSTagFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSTagFilterWithDefaults() *AWSTagFilter {
	this := AWSTagFilter{}
	return &this
}

// GetNamespace returns the Namespace field value if set, zero value otherwise.
func (o *AWSTagFilter) GetNamespace() AWSNamespace {
	if o == nil || o.Namespace == nil {
		var ret AWSNamespace
		return ret
	}
	return *o.Namespace
}

// GetNamespaceOk returns a tuple with the Namespace field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSTagFilter) GetNamespaceOk() (*AWSNamespace, bool) {
	if o == nil || o.Namespace == nil {
		return nil, false
	}
	return o.Namespace, true
}

// HasNamespace returns a boolean if a field has been set.
func (o *AWSTagFilter) HasNamespace() bool {
	if o != nil && o.Namespace != nil {
		return true
	}

	return false
}

// SetNamespace gets a reference to the given AWSNamespace and assigns it to the Namespace field.
func (o *AWSTagFilter) SetNamespace(v AWSNamespace) {
	o.Namespace = &v
}

// GetTagFilterStr returns the TagFilterStr field value if set, zero value otherwise.
func (o *AWSTagFilter) GetTagFilterStr() string {
	if o == nil || o.TagFilterStr == nil {
		var ret string
		return ret
	}
	return *o.TagFilterStr
}

// GetTagFilterStrOk returns a tuple with the TagFilterStr field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSTagFilter) GetTagFilterStrOk() (*string, bool) {
	if o == nil || o.TagFilterStr == nil {
		return nil, false
	}
	return o.TagFilterStr, true
}

// HasTagFilterStr returns a boolean if a field has been set.
func (o *AWSTagFilter) HasTagFilterStr() bool {
	if o != nil && o.TagFilterStr != nil {
		return true
	}

	return false
}

// SetTagFilterStr gets a reference to the given string and assigns it to the TagFilterStr field.
func (o *AWSTagFilter) SetTagFilterStr(v string) {
	o.TagFilterStr = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSTagFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Namespace != nil {
		toSerialize["namespace"] = o.Namespace
	}
	if o.TagFilterStr != nil {
		toSerialize["tag_filter_str"] = o.TagFilterStr
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSTagFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Namespace    *AWSNamespace `json:"namespace,omitempty"`
		TagFilterStr *string       `json:"tag_filter_str,omitempty"`
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
	if v := all.Namespace; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Namespace = all.Namespace
	o.TagFilterStr = all.TagFilterStr
	return nil
}
