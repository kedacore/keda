// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import "github.com/opensearch-project/opensearch-go/v4"

// Inspect represents the struct returned by Inspect() func, its main use is to return the opensearch.Response to the user
type Inspect struct {
	Response *opensearch.Response
}
