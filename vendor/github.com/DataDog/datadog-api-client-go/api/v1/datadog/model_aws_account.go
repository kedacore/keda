// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// AWSAccount Returns the AWS account associated with this integration.
type AWSAccount struct {
	// Your AWS access key ID. Only required if your AWS account is a GovCloud or China account.
	AccessKeyId *string `json:"access_key_id,omitempty"`
	// Your AWS Account ID without dashes.
	AccountId *string `json:"account_id,omitempty"`
	// An object, (in the form `{"namespace1":true/false, "namespace2":true/false}`),
	// that enables or disables metric collection for specific AWS namespaces for this
	// AWS account only.
	AccountSpecificNamespaceRules map[string]bool `json:"account_specific_namespace_rules,omitempty"`
	// Whether Datadog collects cloud security posture management resources from your AWS account. This includes additional resources not covered under the general `resource_collection`.
	CspmResourceCollectionEnabled *bool `json:"cspm_resource_collection_enabled,omitempty"`
	// An array of AWS regions to exclude from metrics collection.
	ExcludedRegions []string `json:"excluded_regions,omitempty"`
	// The array of EC2 tags (in the form `key:value`) defines a filter that Datadog uses when collecting metrics from EC2.
	// Wildcards, such as `?` (for single characters) and `*` (for multiple characters) can also be used.
	// Only hosts that match one of the defined tags
	// will be imported into Datadog. The rest will be ignored.
	// Host matching a given tag can also be excluded by adding `!` before the tag.
	// For example, `env:production,instance-type:c1.*,!region:us-east-1`
	FilterTags []string `json:"filter_tags,omitempty"`
	// Array of tags (in the form `key:value`) to add to all hosts
	// and metrics reporting through this integration.
	HostTags []string `json:"host_tags,omitempty"`
	// Whether Datadog collects metrics for this AWS account.
	MetricsCollectionEnabled *bool `json:"metrics_collection_enabled,omitempty"`
	// Whether Datadog collects a standard set of resources from your AWS account.
	ResourceCollectionEnabled *bool `json:"resource_collection_enabled,omitempty"`
	// Your Datadog role delegation name.
	RoleName *string `json:"role_name,omitempty"`
	// Your AWS secret access key. Only required if your AWS account is a GovCloud or China account.
	SecretAccessKey *string `json:"secret_access_key,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAWSAccount instantiates a new AWSAccount object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAWSAccount() *AWSAccount {
	this := AWSAccount{}
	var cspmResourceCollectionEnabled bool = false
	this.CspmResourceCollectionEnabled = &cspmResourceCollectionEnabled
	var metricsCollectionEnabled bool = true
	this.MetricsCollectionEnabled = &metricsCollectionEnabled
	var resourceCollectionEnabled bool = false
	this.ResourceCollectionEnabled = &resourceCollectionEnabled
	return &this
}

// NewAWSAccountWithDefaults instantiates a new AWSAccount object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAWSAccountWithDefaults() *AWSAccount {
	this := AWSAccount{}
	var cspmResourceCollectionEnabled bool = false
	this.CspmResourceCollectionEnabled = &cspmResourceCollectionEnabled
	var metricsCollectionEnabled bool = true
	this.MetricsCollectionEnabled = &metricsCollectionEnabled
	var resourceCollectionEnabled bool = false
	this.ResourceCollectionEnabled = &resourceCollectionEnabled
	return &this
}

// GetAccessKeyId returns the AccessKeyId field value if set, zero value otherwise.
func (o *AWSAccount) GetAccessKeyId() string {
	if o == nil || o.AccessKeyId == nil {
		var ret string
		return ret
	}
	return *o.AccessKeyId
}

// GetAccessKeyIdOk returns a tuple with the AccessKeyId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetAccessKeyIdOk() (*string, bool) {
	if o == nil || o.AccessKeyId == nil {
		return nil, false
	}
	return o.AccessKeyId, true
}

// HasAccessKeyId returns a boolean if a field has been set.
func (o *AWSAccount) HasAccessKeyId() bool {
	if o != nil && o.AccessKeyId != nil {
		return true
	}

	return false
}

// SetAccessKeyId gets a reference to the given string and assigns it to the AccessKeyId field.
func (o *AWSAccount) SetAccessKeyId(v string) {
	o.AccessKeyId = &v
}

// GetAccountId returns the AccountId field value if set, zero value otherwise.
func (o *AWSAccount) GetAccountId() string {
	if o == nil || o.AccountId == nil {
		var ret string
		return ret
	}
	return *o.AccountId
}

// GetAccountIdOk returns a tuple with the AccountId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetAccountIdOk() (*string, bool) {
	if o == nil || o.AccountId == nil {
		return nil, false
	}
	return o.AccountId, true
}

// HasAccountId returns a boolean if a field has been set.
func (o *AWSAccount) HasAccountId() bool {
	if o != nil && o.AccountId != nil {
		return true
	}

	return false
}

// SetAccountId gets a reference to the given string and assigns it to the AccountId field.
func (o *AWSAccount) SetAccountId(v string) {
	o.AccountId = &v
}

// GetAccountSpecificNamespaceRules returns the AccountSpecificNamespaceRules field value if set, zero value otherwise.
func (o *AWSAccount) GetAccountSpecificNamespaceRules() map[string]bool {
	if o == nil || o.AccountSpecificNamespaceRules == nil {
		var ret map[string]bool
		return ret
	}
	return o.AccountSpecificNamespaceRules
}

// GetAccountSpecificNamespaceRulesOk returns a tuple with the AccountSpecificNamespaceRules field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetAccountSpecificNamespaceRulesOk() (*map[string]bool, bool) {
	if o == nil || o.AccountSpecificNamespaceRules == nil {
		return nil, false
	}
	return &o.AccountSpecificNamespaceRules, true
}

// HasAccountSpecificNamespaceRules returns a boolean if a field has been set.
func (o *AWSAccount) HasAccountSpecificNamespaceRules() bool {
	if o != nil && o.AccountSpecificNamespaceRules != nil {
		return true
	}

	return false
}

// SetAccountSpecificNamespaceRules gets a reference to the given map[string]bool and assigns it to the AccountSpecificNamespaceRules field.
func (o *AWSAccount) SetAccountSpecificNamespaceRules(v map[string]bool) {
	o.AccountSpecificNamespaceRules = v
}

// GetCspmResourceCollectionEnabled returns the CspmResourceCollectionEnabled field value if set, zero value otherwise.
func (o *AWSAccount) GetCspmResourceCollectionEnabled() bool {
	if o == nil || o.CspmResourceCollectionEnabled == nil {
		var ret bool
		return ret
	}
	return *o.CspmResourceCollectionEnabled
}

// GetCspmResourceCollectionEnabledOk returns a tuple with the CspmResourceCollectionEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetCspmResourceCollectionEnabledOk() (*bool, bool) {
	if o == nil || o.CspmResourceCollectionEnabled == nil {
		return nil, false
	}
	return o.CspmResourceCollectionEnabled, true
}

// HasCspmResourceCollectionEnabled returns a boolean if a field has been set.
func (o *AWSAccount) HasCspmResourceCollectionEnabled() bool {
	if o != nil && o.CspmResourceCollectionEnabled != nil {
		return true
	}

	return false
}

// SetCspmResourceCollectionEnabled gets a reference to the given bool and assigns it to the CspmResourceCollectionEnabled field.
func (o *AWSAccount) SetCspmResourceCollectionEnabled(v bool) {
	o.CspmResourceCollectionEnabled = &v
}

// GetExcludedRegions returns the ExcludedRegions field value if set, zero value otherwise.
func (o *AWSAccount) GetExcludedRegions() []string {
	if o == nil || o.ExcludedRegions == nil {
		var ret []string
		return ret
	}
	return o.ExcludedRegions
}

// GetExcludedRegionsOk returns a tuple with the ExcludedRegions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetExcludedRegionsOk() (*[]string, bool) {
	if o == nil || o.ExcludedRegions == nil {
		return nil, false
	}
	return &o.ExcludedRegions, true
}

// HasExcludedRegions returns a boolean if a field has been set.
func (o *AWSAccount) HasExcludedRegions() bool {
	if o != nil && o.ExcludedRegions != nil {
		return true
	}

	return false
}

// SetExcludedRegions gets a reference to the given []string and assigns it to the ExcludedRegions field.
func (o *AWSAccount) SetExcludedRegions(v []string) {
	o.ExcludedRegions = v
}

// GetFilterTags returns the FilterTags field value if set, zero value otherwise.
func (o *AWSAccount) GetFilterTags() []string {
	if o == nil || o.FilterTags == nil {
		var ret []string
		return ret
	}
	return o.FilterTags
}

// GetFilterTagsOk returns a tuple with the FilterTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetFilterTagsOk() (*[]string, bool) {
	if o == nil || o.FilterTags == nil {
		return nil, false
	}
	return &o.FilterTags, true
}

// HasFilterTags returns a boolean if a field has been set.
func (o *AWSAccount) HasFilterTags() bool {
	if o != nil && o.FilterTags != nil {
		return true
	}

	return false
}

// SetFilterTags gets a reference to the given []string and assigns it to the FilterTags field.
func (o *AWSAccount) SetFilterTags(v []string) {
	o.FilterTags = v
}

// GetHostTags returns the HostTags field value if set, zero value otherwise.
func (o *AWSAccount) GetHostTags() []string {
	if o == nil || o.HostTags == nil {
		var ret []string
		return ret
	}
	return o.HostTags
}

// GetHostTagsOk returns a tuple with the HostTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetHostTagsOk() (*[]string, bool) {
	if o == nil || o.HostTags == nil {
		return nil, false
	}
	return &o.HostTags, true
}

// HasHostTags returns a boolean if a field has been set.
func (o *AWSAccount) HasHostTags() bool {
	if o != nil && o.HostTags != nil {
		return true
	}

	return false
}

// SetHostTags gets a reference to the given []string and assigns it to the HostTags field.
func (o *AWSAccount) SetHostTags(v []string) {
	o.HostTags = v
}

// GetMetricsCollectionEnabled returns the MetricsCollectionEnabled field value if set, zero value otherwise.
func (o *AWSAccount) GetMetricsCollectionEnabled() bool {
	if o == nil || o.MetricsCollectionEnabled == nil {
		var ret bool
		return ret
	}
	return *o.MetricsCollectionEnabled
}

// GetMetricsCollectionEnabledOk returns a tuple with the MetricsCollectionEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetMetricsCollectionEnabledOk() (*bool, bool) {
	if o == nil || o.MetricsCollectionEnabled == nil {
		return nil, false
	}
	return o.MetricsCollectionEnabled, true
}

// HasMetricsCollectionEnabled returns a boolean if a field has been set.
func (o *AWSAccount) HasMetricsCollectionEnabled() bool {
	if o != nil && o.MetricsCollectionEnabled != nil {
		return true
	}

	return false
}

// SetMetricsCollectionEnabled gets a reference to the given bool and assigns it to the MetricsCollectionEnabled field.
func (o *AWSAccount) SetMetricsCollectionEnabled(v bool) {
	o.MetricsCollectionEnabled = &v
}

// GetResourceCollectionEnabled returns the ResourceCollectionEnabled field value if set, zero value otherwise.
func (o *AWSAccount) GetResourceCollectionEnabled() bool {
	if o == nil || o.ResourceCollectionEnabled == nil {
		var ret bool
		return ret
	}
	return *o.ResourceCollectionEnabled
}

// GetResourceCollectionEnabledOk returns a tuple with the ResourceCollectionEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetResourceCollectionEnabledOk() (*bool, bool) {
	if o == nil || o.ResourceCollectionEnabled == nil {
		return nil, false
	}
	return o.ResourceCollectionEnabled, true
}

// HasResourceCollectionEnabled returns a boolean if a field has been set.
func (o *AWSAccount) HasResourceCollectionEnabled() bool {
	if o != nil && o.ResourceCollectionEnabled != nil {
		return true
	}

	return false
}

// SetResourceCollectionEnabled gets a reference to the given bool and assigns it to the ResourceCollectionEnabled field.
func (o *AWSAccount) SetResourceCollectionEnabled(v bool) {
	o.ResourceCollectionEnabled = &v
}

// GetRoleName returns the RoleName field value if set, zero value otherwise.
func (o *AWSAccount) GetRoleName() string {
	if o == nil || o.RoleName == nil {
		var ret string
		return ret
	}
	return *o.RoleName
}

// GetRoleNameOk returns a tuple with the RoleName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetRoleNameOk() (*string, bool) {
	if o == nil || o.RoleName == nil {
		return nil, false
	}
	return o.RoleName, true
}

// HasRoleName returns a boolean if a field has been set.
func (o *AWSAccount) HasRoleName() bool {
	if o != nil && o.RoleName != nil {
		return true
	}

	return false
}

// SetRoleName gets a reference to the given string and assigns it to the RoleName field.
func (o *AWSAccount) SetRoleName(v string) {
	o.RoleName = &v
}

// GetSecretAccessKey returns the SecretAccessKey field value if set, zero value otherwise.
func (o *AWSAccount) GetSecretAccessKey() string {
	if o == nil || o.SecretAccessKey == nil {
		var ret string
		return ret
	}
	return *o.SecretAccessKey
}

// GetSecretAccessKeyOk returns a tuple with the SecretAccessKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AWSAccount) GetSecretAccessKeyOk() (*string, bool) {
	if o == nil || o.SecretAccessKey == nil {
		return nil, false
	}
	return o.SecretAccessKey, true
}

// HasSecretAccessKey returns a boolean if a field has been set.
func (o *AWSAccount) HasSecretAccessKey() bool {
	if o != nil && o.SecretAccessKey != nil {
		return true
	}

	return false
}

// SetSecretAccessKey gets a reference to the given string and assigns it to the SecretAccessKey field.
func (o *AWSAccount) SetSecretAccessKey(v string) {
	o.SecretAccessKey = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AWSAccount) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AccessKeyId != nil {
		toSerialize["access_key_id"] = o.AccessKeyId
	}
	if o.AccountId != nil {
		toSerialize["account_id"] = o.AccountId
	}
	if o.AccountSpecificNamespaceRules != nil {
		toSerialize["account_specific_namespace_rules"] = o.AccountSpecificNamespaceRules
	}
	if o.CspmResourceCollectionEnabled != nil {
		toSerialize["cspm_resource_collection_enabled"] = o.CspmResourceCollectionEnabled
	}
	if o.ExcludedRegions != nil {
		toSerialize["excluded_regions"] = o.ExcludedRegions
	}
	if o.FilterTags != nil {
		toSerialize["filter_tags"] = o.FilterTags
	}
	if o.HostTags != nil {
		toSerialize["host_tags"] = o.HostTags
	}
	if o.MetricsCollectionEnabled != nil {
		toSerialize["metrics_collection_enabled"] = o.MetricsCollectionEnabled
	}
	if o.ResourceCollectionEnabled != nil {
		toSerialize["resource_collection_enabled"] = o.ResourceCollectionEnabled
	}
	if o.RoleName != nil {
		toSerialize["role_name"] = o.RoleName
	}
	if o.SecretAccessKey != nil {
		toSerialize["secret_access_key"] = o.SecretAccessKey
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AWSAccount) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AccessKeyId                   *string         `json:"access_key_id,omitempty"`
		AccountId                     *string         `json:"account_id,omitempty"`
		AccountSpecificNamespaceRules map[string]bool `json:"account_specific_namespace_rules,omitempty"`
		CspmResourceCollectionEnabled *bool           `json:"cspm_resource_collection_enabled,omitempty"`
		ExcludedRegions               []string        `json:"excluded_regions,omitempty"`
		FilterTags                    []string        `json:"filter_tags,omitempty"`
		HostTags                      []string        `json:"host_tags,omitempty"`
		MetricsCollectionEnabled      *bool           `json:"metrics_collection_enabled,omitempty"`
		ResourceCollectionEnabled     *bool           `json:"resource_collection_enabled,omitempty"`
		RoleName                      *string         `json:"role_name,omitempty"`
		SecretAccessKey               *string         `json:"secret_access_key,omitempty"`
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
	o.AccessKeyId = all.AccessKeyId
	o.AccountId = all.AccountId
	o.AccountSpecificNamespaceRules = all.AccountSpecificNamespaceRules
	o.CspmResourceCollectionEnabled = all.CspmResourceCollectionEnabled
	o.ExcludedRegions = all.ExcludedRegions
	o.FilterTags = all.FilterTags
	o.HostTags = all.HostTags
	o.MetricsCollectionEnabled = all.MetricsCollectionEnabled
	o.ResourceCollectionEnabled = all.ResourceCollectionEnabled
	o.RoleName = all.RoleName
	o.SecretAccessKey = all.SecretAccessKey
	return nil
}
