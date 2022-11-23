// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// Organization Create, edit, and manage organizations.
type Organization struct {
	// A JSON array of billing type.
	// Deprecated
	Billing *OrganizationBilling `json:"billing,omitempty"`
	// Date of the organization creation.
	Created *string `json:"created,omitempty"`
	// Description of the organization.
	Description *string `json:"description,omitempty"`
	// The name of the new child-organization, limited to 32 characters.
	Name *string `json:"name,omitempty"`
	// The `public_id` of the organization you are operating within.
	PublicId *string `json:"public_id,omitempty"`
	// A JSON array of settings.
	Settings *OrganizationSettings `json:"settings,omitempty"`
	// Subscription definition.
	// Deprecated
	Subscription *OrganizationSubscription `json:"subscription,omitempty"`
	// Only available for MSP customers. Allows child organizations to be created on a trial plan.
	Trial *bool `json:"trial,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewOrganization instantiates a new Organization object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganization() *Organization {
	this := Organization{}
	return &this
}

// NewOrganizationWithDefaults instantiates a new Organization object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationWithDefaults() *Organization {
	this := Organization{}
	return &this
}

// GetBilling returns the Billing field value if set, zero value otherwise.
// Deprecated
func (o *Organization) GetBilling() OrganizationBilling {
	if o == nil || o.Billing == nil {
		var ret OrganizationBilling
		return ret
	}
	return *o.Billing
}

// GetBillingOk returns a tuple with the Billing field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *Organization) GetBillingOk() (*OrganizationBilling, bool) {
	if o == nil || o.Billing == nil {
		return nil, false
	}
	return o.Billing, true
}

// HasBilling returns a boolean if a field has been set.
func (o *Organization) HasBilling() bool {
	if o != nil && o.Billing != nil {
		return true
	}

	return false
}

// SetBilling gets a reference to the given OrganizationBilling and assigns it to the Billing field.
// Deprecated
func (o *Organization) SetBilling(v OrganizationBilling) {
	o.Billing = &v
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *Organization) GetCreated() string {
	if o == nil || o.Created == nil {
		var ret string
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Organization) GetCreatedOk() (*string, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *Organization) HasCreated() bool {
	if o != nil && o.Created != nil {
		return true
	}

	return false
}

// SetCreated gets a reference to the given string and assigns it to the Created field.
func (o *Organization) SetCreated(v string) {
	o.Created = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *Organization) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Organization) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *Organization) HasDescription() bool {
	if o != nil && o.Description != nil {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *Organization) SetDescription(v string) {
	o.Description = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *Organization) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Organization) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *Organization) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *Organization) SetName(v string) {
	o.Name = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *Organization) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Organization) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *Organization) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *Organization) SetPublicId(v string) {
	o.PublicId = &v
}

// GetSettings returns the Settings field value if set, zero value otherwise.
func (o *Organization) GetSettings() OrganizationSettings {
	if o == nil || o.Settings == nil {
		var ret OrganizationSettings
		return ret
	}
	return *o.Settings
}

// GetSettingsOk returns a tuple with the Settings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Organization) GetSettingsOk() (*OrganizationSettings, bool) {
	if o == nil || o.Settings == nil {
		return nil, false
	}
	return o.Settings, true
}

// HasSettings returns a boolean if a field has been set.
func (o *Organization) HasSettings() bool {
	if o != nil && o.Settings != nil {
		return true
	}

	return false
}

// SetSettings gets a reference to the given OrganizationSettings and assigns it to the Settings field.
func (o *Organization) SetSettings(v OrganizationSettings) {
	o.Settings = &v
}

// GetSubscription returns the Subscription field value if set, zero value otherwise.
// Deprecated
func (o *Organization) GetSubscription() OrganizationSubscription {
	if o == nil || o.Subscription == nil {
		var ret OrganizationSubscription
		return ret
	}
	return *o.Subscription
}

// GetSubscriptionOk returns a tuple with the Subscription field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *Organization) GetSubscriptionOk() (*OrganizationSubscription, bool) {
	if o == nil || o.Subscription == nil {
		return nil, false
	}
	return o.Subscription, true
}

// HasSubscription returns a boolean if a field has been set.
func (o *Organization) HasSubscription() bool {
	if o != nil && o.Subscription != nil {
		return true
	}

	return false
}

// SetSubscription gets a reference to the given OrganizationSubscription and assigns it to the Subscription field.
// Deprecated
func (o *Organization) SetSubscription(v OrganizationSubscription) {
	o.Subscription = &v
}

// GetTrial returns the Trial field value if set, zero value otherwise.
func (o *Organization) GetTrial() bool {
	if o == nil || o.Trial == nil {
		var ret bool
		return ret
	}
	return *o.Trial
}

// GetTrialOk returns a tuple with the Trial field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Organization) GetTrialOk() (*bool, bool) {
	if o == nil || o.Trial == nil {
		return nil, false
	}
	return o.Trial, true
}

// HasTrial returns a boolean if a field has been set.
func (o *Organization) HasTrial() bool {
	if o != nil && o.Trial != nil {
		return true
	}

	return false
}

// SetTrial gets a reference to the given bool and assigns it to the Trial field.
func (o *Organization) SetTrial(v bool) {
	o.Trial = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o Organization) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Billing != nil {
		toSerialize["billing"] = o.Billing
	}
	if o.Created != nil {
		toSerialize["created"] = o.Created
	}
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.Settings != nil {
		toSerialize["settings"] = o.Settings
	}
	if o.Subscription != nil {
		toSerialize["subscription"] = o.Subscription
	}
	if o.Trial != nil {
		toSerialize["trial"] = o.Trial
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *Organization) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Billing      *OrganizationBilling      `json:"billing,omitempty"`
		Created      *string                   `json:"created,omitempty"`
		Description  *string                   `json:"description,omitempty"`
		Name         *string                   `json:"name,omitempty"`
		PublicId     *string                   `json:"public_id,omitempty"`
		Settings     *OrganizationSettings     `json:"settings,omitempty"`
		Subscription *OrganizationSubscription `json:"subscription,omitempty"`
		Trial        *bool                     `json:"trial,omitempty"`
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
	if all.Billing != nil && all.Billing.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Billing = all.Billing
	o.Created = all.Created
	o.Description = all.Description
	o.Name = all.Name
	o.PublicId = all.PublicId
	if all.Settings != nil && all.Settings.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Settings = all.Settings
	if all.Subscription != nil && all.Subscription.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Subscription = all.Subscription
	o.Trial = all.Trial
	return nil
}
