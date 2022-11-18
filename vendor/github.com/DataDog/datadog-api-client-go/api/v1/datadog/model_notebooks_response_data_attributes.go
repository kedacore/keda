// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
	"time"
)

// NotebooksResponseDataAttributes The attributes of a notebook in get all response.
type NotebooksResponseDataAttributes struct {
	// Attributes of user object returned by the API.
	Author *NotebookAuthor `json:"author,omitempty"`
	// List of cells to display in the notebook.
	Cells []NotebookCellResponse `json:"cells,omitempty"`
	// UTC time stamp for when the notebook was created.
	Created *time.Time `json:"created,omitempty"`
	// Metadata associated with the notebook.
	Metadata *NotebookMetadata `json:"metadata,omitempty"`
	// UTC time stamp for when the notebook was last modified.
	Modified *time.Time `json:"modified,omitempty"`
	// The name of the notebook.
	Name string `json:"name"`
	// Publication status of the notebook. For now, always "published".
	Status *NotebookStatus `json:"status,omitempty"`
	// Notebook global timeframe.
	Time *NotebookGlobalTime `json:"time,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebooksResponseDataAttributes instantiates a new NotebooksResponseDataAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebooksResponseDataAttributes(name string) *NotebooksResponseDataAttributes {
	this := NotebooksResponseDataAttributes{}
	this.Name = name
	var status NotebookStatus = NOTEBOOKSTATUS_PUBLISHED
	this.Status = &status
	return &this
}

// NewNotebooksResponseDataAttributesWithDefaults instantiates a new NotebooksResponseDataAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebooksResponseDataAttributesWithDefaults() *NotebooksResponseDataAttributes {
	this := NotebooksResponseDataAttributes{}
	var status NotebookStatus = NOTEBOOKSTATUS_PUBLISHED
	this.Status = &status
	return &this
}

// GetAuthor returns the Author field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetAuthor() NotebookAuthor {
	if o == nil || o.Author == nil {
		var ret NotebookAuthor
		return ret
	}
	return *o.Author
}

// GetAuthorOk returns a tuple with the Author field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetAuthorOk() (*NotebookAuthor, bool) {
	if o == nil || o.Author == nil {
		return nil, false
	}
	return o.Author, true
}

// HasAuthor returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasAuthor() bool {
	if o != nil && o.Author != nil {
		return true
	}

	return false
}

// SetAuthor gets a reference to the given NotebookAuthor and assigns it to the Author field.
func (o *NotebooksResponseDataAttributes) SetAuthor(v NotebookAuthor) {
	o.Author = &v
}

// GetCells returns the Cells field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetCells() []NotebookCellResponse {
	if o == nil || o.Cells == nil {
		var ret []NotebookCellResponse
		return ret
	}
	return o.Cells
}

// GetCellsOk returns a tuple with the Cells field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetCellsOk() (*[]NotebookCellResponse, bool) {
	if o == nil || o.Cells == nil {
		return nil, false
	}
	return &o.Cells, true
}

// HasCells returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasCells() bool {
	if o != nil && o.Cells != nil {
		return true
	}

	return false
}

// SetCells gets a reference to the given []NotebookCellResponse and assigns it to the Cells field.
func (o *NotebooksResponseDataAttributes) SetCells(v []NotebookCellResponse) {
	o.Cells = v
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetCreated() time.Time {
	if o == nil || o.Created == nil {
		var ret time.Time
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetCreatedOk() (*time.Time, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasCreated() bool {
	if o != nil && o.Created != nil {
		return true
	}

	return false
}

// SetCreated gets a reference to the given time.Time and assigns it to the Created field.
func (o *NotebooksResponseDataAttributes) SetCreated(v time.Time) {
	o.Created = &v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetMetadata() NotebookMetadata {
	if o == nil || o.Metadata == nil {
		var ret NotebookMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetMetadataOk() (*NotebookMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given NotebookMetadata and assigns it to the Metadata field.
func (o *NotebooksResponseDataAttributes) SetMetadata(v NotebookMetadata) {
	o.Metadata = &v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetModified() time.Time {
	if o == nil || o.Modified == nil {
		var ret time.Time
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetModifiedOk() (*time.Time, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasModified() bool {
	if o != nil && o.Modified != nil {
		return true
	}

	return false
}

// SetModified gets a reference to the given time.Time and assigns it to the Modified field.
func (o *NotebooksResponseDataAttributes) SetModified(v time.Time) {
	o.Modified = &v
}

// GetName returns the Name field value.
func (o *NotebooksResponseDataAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *NotebooksResponseDataAttributes) SetName(v string) {
	o.Name = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetStatus() NotebookStatus {
	if o == nil || o.Status == nil {
		var ret NotebookStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetStatusOk() (*NotebookStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given NotebookStatus and assigns it to the Status field.
func (o *NotebooksResponseDataAttributes) SetStatus(v NotebookStatus) {
	o.Status = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *NotebooksResponseDataAttributes) GetTime() NotebookGlobalTime {
	if o == nil || o.Time == nil {
		var ret NotebookGlobalTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebooksResponseDataAttributes) GetTimeOk() (*NotebookGlobalTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *NotebooksResponseDataAttributes) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given NotebookGlobalTime and assigns it to the Time field.
func (o *NotebooksResponseDataAttributes) SetTime(v NotebookGlobalTime) {
	o.Time = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebooksResponseDataAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Author != nil {
		toSerialize["author"] = o.Author
	}
	if o.Cells != nil {
		toSerialize["cells"] = o.Cells
	}
	if o.Created != nil {
		if o.Created.Nanosecond() == 0 {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	if o.Modified != nil {
		if o.Modified.Nanosecond() == 0 {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	toSerialize["name"] = o.Name
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Time != nil {
		toSerialize["time"] = o.Time
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebooksResponseDataAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		Author   *NotebookAuthor        `json:"author,omitempty"`
		Cells    []NotebookCellResponse `json:"cells,omitempty"`
		Created  *time.Time             `json:"created,omitempty"`
		Metadata *NotebookMetadata      `json:"metadata,omitempty"`
		Modified *time.Time             `json:"modified,omitempty"`
		Name     string                 `json:"name"`
		Status   *NotebookStatus        `json:"status,omitempty"`
		Time     *NotebookGlobalTime    `json:"time,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Author != nil && all.Author.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Author = all.Author
	o.Cells = all.Cells
	o.Created = all.Created
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Modified = all.Modified
	o.Name = all.Name
	o.Status = all.Status
	o.Time = all.Time
	return nil
}
