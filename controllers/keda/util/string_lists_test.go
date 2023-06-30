/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"reflect"
	"testing"
)

func TestAppendString(t *testing.T) {
	testData := []struct {
		name   string
		from   string
		append string
		sep    string
		exp    string
	}{
		{"success", "viewer,editor", "owner", ",", "viewer,editor,owner"},
		{"single_success", "viewer", "owner", ",", "viewer,owner"},
		{"exist", "viewer,editor,owner", "editor", ",", "viewer,editor,owner"},
		{"no_separator", "viewer,editor", "owner", "", "viewer,editorowner"},
		{"space_separator", "viewer,editor", "owner", " ", "viewer,editor owner"},
		{"diff_separator", "viewer,editor", "owner", ":", "viewer,editor:owner"},
		{"no_from_str", "", "owner", ",", "owner"},
		{"no_append_str", "viewer,editor", "", ",", "viewer,editor"},
	}

	for _, tt := range testData {
		got := AppendIntoString(tt.from, tt.append, tt.sep)

		if !reflect.DeepEqual(tt.exp, got) {
			t.Errorf("Expected %v but got %v\n", tt.exp, got)
		}
	}
}

func TestRemoveFromString(t *testing.T) {
	testData := []struct {
		name   string
		from   string
		delete string
		sep    string
		exp    string
	}{
		{"success", "viewer,editor,owner", "owner", ",", "viewer,editor"},
		{"no_exist_success", "viewer", "owner", ",", "viewer"},
		{"no_separator", "viewer,editor,owner", "owner", "", "viewer,editor,owner"},
		{"space_separator", "viewer editor owner", "editor", " ", "viewer owner"},
		{"diff_separator", "viewer,editor,owner", "editor", ":", "viewer,editor,owner"},
		{"no_from_str", "", "owner", ",", ""},
		{"no_delete_str", "viewer,editor", "", ",", "viewer,editor"},
	}

	for _, tt := range testData {
		got := RemoveFromString(tt.from, tt.delete, tt.sep)

		if !reflect.DeepEqual(tt.exp, got) {
			t.Errorf("Expected %v but got %v\n", tt.exp, got)
		}
	}
}
