/*
Copyright 2020 The Knative Authors

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

package kmeta

import "knative.dev/pkg/kmap"

// CopyMap makes a copy of the map.
// Deprecated: use kmap.Copy
var CopyMap = kmap.Copy

// UnionMaps returns a map constructed from the union of input maps.
// where values from latter maps win.
// Deprecated: use kmap.Union
var UnionMaps = kmap.Union

// FilterMap creates a copy of the provided map, filtering out the elements
// that match `filter`.
// nil `filter` is accepted.
// Deprecated: use kmap.Filter
var FilterMap = kmap.Filter
