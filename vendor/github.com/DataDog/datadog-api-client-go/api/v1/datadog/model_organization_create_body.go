// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// OrganizationCreateBody Object describing an organization to create.
type OrganizationCreateBody struct {
	// A JSON array of billing type.
	// Deprecated
	Billing *OrganizationBilling `json:"billing,omitempty"`
	// The name of the new child-organization, limited to 32 characters.
	Name string `json:"name"`
	// Subscription definition.
	// Deprecated
	Subscription *OrganizationSubscription `json:"subscription,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewOrganizationCreateBody instantiates a new OrganizationCreateBody object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganizationCreateBody(name string) *OrganizationCreateBody {
	this := OrganizationCreateBody{}
	this.Name = name
	return &this
}

// NewOrganizationCreateBodyWithDefaults instantiates a new OrganizationCreateBody object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationCreateBodyWithDefaults() *OrganizationCreateBody {
	this := OrganizationCreateBody{}
	return &this
}

// GetBilling returns the Billing field value if set, zero value otherwise.
// Deprecated
func (o *OrganizationCreateBody) GetBilling() OrganizationBilling {
	if o == nil || o.Billing == nil {
		var ret OrganizationBilling
		return ret
	}
	return *o.Billing
}

// GetBillingOk returns a tuple with the Billing field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *OrganizationCreateBody) GetBillingOk() (*OrganizationBilling, bool) {
	if o == nil || o.Billing == nil {
		return nil, false
	}
	return o.Billing, true
}

// HasBilling returns a boolean if a field has been set.
func (o *OrganizationCreateBody) HasBilling() bool {
	if o != nil && o.Billing != nil {
		return true
	}

	return false
}

// SetBilling gets a reference to the given OrganizationBilling and assigns it to the Billing field.
// Deprecated
func (o *OrganizationCreateBody) SetBilling(v OrganizationBilling) {
	o.Billing = &v
}

// GetName returns the Name field value.
func (o *OrganizationCreateBody) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *OrganizationCreateBody) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *OrganizationCreateBody) SetName(v string) {
	o.Name = v
}

// GetSubscription returns the Subscription field value if set, zero value otherwise.
// Deprecated
func (o *OrganizationCreateBody) GetSubscription() OrganizationSubscription {
	if o == nil || o.Subscription == nil {
		var ret OrganizationSubscription
		return ret
	}
	return *o.Subscription
}

// GetSubscriptionOk returns a tuple with the Subscription field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *OrganizationCreateBody) GetSubscriptionOk() (*OrganizationSubscription, bool) {
	if o == nil || o.Subscription == nil {
		return nil, false
	}
	return o.Subscription, true
}

// HasSubscription returns a boolean if a field has been set.
func (o *OrganizationCreateBody) HasSubscription() bool {
	if o != nil && o.Subscription != nil {
		return true
	}

	return false
}

// SetSubscription gets a reference to the given OrganizationSubscription and assigns it to the Subscription field.
// Deprecated
func (o *OrganizationCreateBody) SetSubscription(v OrganizationSubscription) {
	o.Subscription = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OrganizationCreateBody) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Billing != nil {
		toSerialize["billing"] = o.Billing
	}
	toSerialize["name"] = o.Name
	if o.Subscription != nil {
		toSerialize["subscription"] = o.Subscription
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OrganizationCreateBody) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		Billing      *OrganizationBilling      `json:"billing,omitempty"`
		Name         string                    `json:"name"`
		Subscription *OrganizationSubscription `json:"subscription,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if all.Billing != nil && all.Billing.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Billing = all.Billing
	o.Name = all.Name
	if all.Subscription != nil && all.Subscription.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Subscription = all.Subscription
	return nil
}
