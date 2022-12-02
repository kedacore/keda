// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookCreateDataAttributes The data attributes of a notebook.
type NotebookCreateDataAttributes struct {
	// List of cells to display in the notebook.
	Cells []NotebookCellCreateRequest `json:"cells"`
	// Metadata associated with the notebook.
	Metadata *NotebookMetadata `json:"metadata,omitempty"`
	// The name of the notebook.
	Name string `json:"name"`
	// Publication status of the notebook. For now, always "published".
	Status *NotebookStatus `json:"status,omitempty"`
	// Notebook global timeframe.
	Time NotebookGlobalTime `json:"time"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookCreateDataAttributes instantiates a new NotebookCreateDataAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookCreateDataAttributes(cells []NotebookCellCreateRequest, name string, time NotebookGlobalTime) *NotebookCreateDataAttributes {
	this := NotebookCreateDataAttributes{}
	this.Cells = cells
	this.Name = name
	var status NotebookStatus = NOTEBOOKSTATUS_PUBLISHED
	this.Status = &status
	this.Time = time
	return &this
}

// NewNotebookCreateDataAttributesWithDefaults instantiates a new NotebookCreateDataAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookCreateDataAttributesWithDefaults() *NotebookCreateDataAttributes {
	this := NotebookCreateDataAttributes{}
	var status NotebookStatus = NOTEBOOKSTATUS_PUBLISHED
	this.Status = &status
	return &this
}

// GetCells returns the Cells field value.
func (o *NotebookCreateDataAttributes) GetCells() []NotebookCellCreateRequest {
	if o == nil {
		var ret []NotebookCellCreateRequest
		return ret
	}
	return o.Cells
}

// GetCellsOk returns a tuple with the Cells field value
// and a boolean to check if the value has been set.
func (o *NotebookCreateDataAttributes) GetCellsOk() (*[]NotebookCellCreateRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Cells, true
}

// SetCells sets field value.
func (o *NotebookCreateDataAttributes) SetCells(v []NotebookCellCreateRequest) {
	o.Cells = v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *NotebookCreateDataAttributes) GetMetadata() NotebookMetadata {
	if o == nil || o.Metadata == nil {
		var ret NotebookMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebookCreateDataAttributes) GetMetadataOk() (*NotebookMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *NotebookCreateDataAttributes) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given NotebookMetadata and assigns it to the Metadata field.
func (o *NotebookCreateDataAttributes) SetMetadata(v NotebookMetadata) {
	o.Metadata = &v
}

// GetName returns the Name field value.
func (o *NotebookCreateDataAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *NotebookCreateDataAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *NotebookCreateDataAttributes) SetName(v string) {
	o.Name = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *NotebookCreateDataAttributes) GetStatus() NotebookStatus {
	if o == nil || o.Status == nil {
		var ret NotebookStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebookCreateDataAttributes) GetStatusOk() (*NotebookStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *NotebookCreateDataAttributes) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given NotebookStatus and assigns it to the Status field.
func (o *NotebookCreateDataAttributes) SetStatus(v NotebookStatus) {
	o.Status = &v
}

// GetTime returns the Time field value.
func (o *NotebookCreateDataAttributes) GetTime() NotebookGlobalTime {
	if o == nil {
		var ret NotebookGlobalTime
		return ret
	}
	return o.Time
}

// GetTimeOk returns a tuple with the Time field value
// and a boolean to check if the value has been set.
func (o *NotebookCreateDataAttributes) GetTimeOk() (*NotebookGlobalTime, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Time, true
}

// SetTime sets field value.
func (o *NotebookCreateDataAttributes) SetTime(v NotebookGlobalTime) {
	o.Time = v
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookCreateDataAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["cells"] = o.Cells
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	toSerialize["name"] = o.Name
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	toSerialize["time"] = o.Time

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookCreateDataAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Cells *[]NotebookCellCreateRequest `json:"cells"`
		Name  *string                      `json:"name"`
		Time  *NotebookGlobalTime          `json:"time"`
	}{}
	all := struct {
		Cells    []NotebookCellCreateRequest `json:"cells"`
		Metadata *NotebookMetadata           `json:"metadata,omitempty"`
		Name     string                      `json:"name"`
		Status   *NotebookStatus             `json:"status,omitempty"`
		Time     NotebookGlobalTime          `json:"time"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Cells == nil {
		return fmt.Errorf("Required field cells missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Cells = all.Cells
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Name = all.Name
	o.Status = all.Status
	o.Time = all.Time
	return nil
}
