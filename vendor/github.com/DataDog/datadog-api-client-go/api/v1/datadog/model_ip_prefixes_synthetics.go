// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// IPPrefixesSynthetics Available prefix information for the Synthetics endpoints.
type IPPrefixesSynthetics struct {
	// List of IPv4 prefixes.
	PrefixesIpv4 []string `json:"prefixes_ipv4,omitempty"`
	// List of IPv4 prefixes by location.
	PrefixesIpv4ByLocation map[string][]string `json:"prefixes_ipv4_by_location,omitempty"`
	// List of IPv6 prefixes.
	PrefixesIpv6 []string `json:"prefixes_ipv6,omitempty"`
	// List of IPv6 prefixes by location.
	PrefixesIpv6ByLocation map[string][]string `json:"prefixes_ipv6_by_location,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewIPPrefixesSynthetics instantiates a new IPPrefixesSynthetics object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIPPrefixesSynthetics() *IPPrefixesSynthetics {
	this := IPPrefixesSynthetics{}
	return &this
}

// NewIPPrefixesSyntheticsWithDefaults instantiates a new IPPrefixesSynthetics object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIPPrefixesSyntheticsWithDefaults() *IPPrefixesSynthetics {
	this := IPPrefixesSynthetics{}
	return &this
}

// GetPrefixesIpv4 returns the PrefixesIpv4 field value if set, zero value otherwise.
func (o *IPPrefixesSynthetics) GetPrefixesIpv4() []string {
	if o == nil || o.PrefixesIpv4 == nil {
		var ret []string
		return ret
	}
	return o.PrefixesIpv4
}

// GetPrefixesIpv4Ok returns a tuple with the PrefixesIpv4 field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPPrefixesSynthetics) GetPrefixesIpv4Ok() (*[]string, bool) {
	if o == nil || o.PrefixesIpv4 == nil {
		return nil, false
	}
	return &o.PrefixesIpv4, true
}

// HasPrefixesIpv4 returns a boolean if a field has been set.
func (o *IPPrefixesSynthetics) HasPrefixesIpv4() bool {
	if o != nil && o.PrefixesIpv4 != nil {
		return true
	}

	return false
}

// SetPrefixesIpv4 gets a reference to the given []string and assigns it to the PrefixesIpv4 field.
func (o *IPPrefixesSynthetics) SetPrefixesIpv4(v []string) {
	o.PrefixesIpv4 = v
}

// GetPrefixesIpv4ByLocation returns the PrefixesIpv4ByLocation field value if set, zero value otherwise.
func (o *IPPrefixesSynthetics) GetPrefixesIpv4ByLocation() map[string][]string {
	if o == nil || o.PrefixesIpv4ByLocation == nil {
		var ret map[string][]string
		return ret
	}
	return o.PrefixesIpv4ByLocation
}

// GetPrefixesIpv4ByLocationOk returns a tuple with the PrefixesIpv4ByLocation field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPPrefixesSynthetics) GetPrefixesIpv4ByLocationOk() (*map[string][]string, bool) {
	if o == nil || o.PrefixesIpv4ByLocation == nil {
		return nil, false
	}
	return &o.PrefixesIpv4ByLocation, true
}

// HasPrefixesIpv4ByLocation returns a boolean if a field has been set.
func (o *IPPrefixesSynthetics) HasPrefixesIpv4ByLocation() bool {
	if o != nil && o.PrefixesIpv4ByLocation != nil {
		return true
	}

	return false
}

// SetPrefixesIpv4ByLocation gets a reference to the given map[string][]string and assigns it to the PrefixesIpv4ByLocation field.
func (o *IPPrefixesSynthetics) SetPrefixesIpv4ByLocation(v map[string][]string) {
	o.PrefixesIpv4ByLocation = v
}

// GetPrefixesIpv6 returns the PrefixesIpv6 field value if set, zero value otherwise.
func (o *IPPrefixesSynthetics) GetPrefixesIpv6() []string {
	if o == nil || o.PrefixesIpv6 == nil {
		var ret []string
		return ret
	}
	return o.PrefixesIpv6
}

// GetPrefixesIpv6Ok returns a tuple with the PrefixesIpv6 field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPPrefixesSynthetics) GetPrefixesIpv6Ok() (*[]string, bool) {
	if o == nil || o.PrefixesIpv6 == nil {
		return nil, false
	}
	return &o.PrefixesIpv6, true
}

// HasPrefixesIpv6 returns a boolean if a field has been set.
func (o *IPPrefixesSynthetics) HasPrefixesIpv6() bool {
	if o != nil && o.PrefixesIpv6 != nil {
		return true
	}

	return false
}

// SetPrefixesIpv6 gets a reference to the given []string and assigns it to the PrefixesIpv6 field.
func (o *IPPrefixesSynthetics) SetPrefixesIpv6(v []string) {
	o.PrefixesIpv6 = v
}

// GetPrefixesIpv6ByLocation returns the PrefixesIpv6ByLocation field value if set, zero value otherwise.
func (o *IPPrefixesSynthetics) GetPrefixesIpv6ByLocation() map[string][]string {
	if o == nil || o.PrefixesIpv6ByLocation == nil {
		var ret map[string][]string
		return ret
	}
	return o.PrefixesIpv6ByLocation
}

// GetPrefixesIpv6ByLocationOk returns a tuple with the PrefixesIpv6ByLocation field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPPrefixesSynthetics) GetPrefixesIpv6ByLocationOk() (*map[string][]string, bool) {
	if o == nil || o.PrefixesIpv6ByLocation == nil {
		return nil, false
	}
	return &o.PrefixesIpv6ByLocation, true
}

// HasPrefixesIpv6ByLocation returns a boolean if a field has been set.
func (o *IPPrefixesSynthetics) HasPrefixesIpv6ByLocation() bool {
	if o != nil && o.PrefixesIpv6ByLocation != nil {
		return true
	}

	return false
}

// SetPrefixesIpv6ByLocation gets a reference to the given map[string][]string and assigns it to the PrefixesIpv6ByLocation field.
func (o *IPPrefixesSynthetics) SetPrefixesIpv6ByLocation(v map[string][]string) {
	o.PrefixesIpv6ByLocation = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IPPrefixesSynthetics) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.PrefixesIpv4 != nil {
		toSerialize["prefixes_ipv4"] = o.PrefixesIpv4
	}
	if o.PrefixesIpv4ByLocation != nil {
		toSerialize["prefixes_ipv4_by_location"] = o.PrefixesIpv4ByLocation
	}
	if o.PrefixesIpv6 != nil {
		toSerialize["prefixes_ipv6"] = o.PrefixesIpv6
	}
	if o.PrefixesIpv6ByLocation != nil {
		toSerialize["prefixes_ipv6_by_location"] = o.PrefixesIpv6ByLocation
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IPPrefixesSynthetics) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		PrefixesIpv4           []string            `json:"prefixes_ipv4,omitempty"`
		PrefixesIpv4ByLocation map[string][]string `json:"prefixes_ipv4_by_location,omitempty"`
		PrefixesIpv6           []string            `json:"prefixes_ipv6,omitempty"`
		PrefixesIpv6ByLocation map[string][]string `json:"prefixes_ipv6_by_location,omitempty"`
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
	o.PrefixesIpv4 = all.PrefixesIpv4
	o.PrefixesIpv4ByLocation = all.PrefixesIpv4ByLocation
	o.PrefixesIpv6 = all.PrefixesIpv6
	o.PrefixesIpv6ByLocation = all.PrefixesIpv6ByLocation
	return nil
}
