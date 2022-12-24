//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Adam Janikowski
//

package driver

import (
	"context"
	"path"

	"github.com/arangodb/go-velocypack"
)

// RevisionUInt64 is representation of '_rev' string value as an uint64 number
type RevisionUInt64 uint64

// RevisionMinMax is an array of two Revisions which create range of them
type RevisionMinMax [2]RevisionUInt64

// Revisions is a slice of Revisions
type Revisions []RevisionUInt64

type RevisionRanges struct {
	Ranges []Revisions    `json:"ranges"`
	Resume RevisionUInt64 `json:"resume,string" velocypack:"resume"`
}

// RevisionTreeNode is a leaf in Merkle tree with hashed Revisions and with count of documents in the leaf
type RevisionTreeNode struct {
	Hash  uint64 `json:"hash"`
	Count uint64 `json:"count,int"`
}

// RevisionTree is a list of Revisions in a Merkle tree
type RevisionTree struct {
	Version         int                `json:"version"`
	MaxDepth        int                `json:"maxDepth"`
	RangeMin        RevisionUInt64     `json:"rangeMin,string" velocypack:"rangeMin"`
	RangeMax        RevisionUInt64     `json:"rangeMax,string" velocypack:"rangeMax"`
	InitialRangeMin RevisionUInt64     `json:"initialRangeMin,string" velocypack:"initialRangeMin"`
	Count           uint64             `json:"count,int"`
	Hash            uint64             `json:"hash"`
	Nodes           []RevisionTreeNode `json:"nodes"`
}

var (
	revisionEncodingTable = [64]byte{'-', '_', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N',
		'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k',
		'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7',
		'8', '9'}
	revisionDecodingTable = [256]byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //   0 - 15
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //  16 - 31
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //  32 - 47 (here is the '-' on 45 place)
		54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 0, 0, 0, 0, 0, 0, //  48 - 63
		0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, //  64 - 79
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 0, 0, 0, 0, 1, //  80 - 95
		0, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, //  96 - 111
		43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 0, 0, 0, 0, 0, // 112 - 127
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 128 - 143
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 144 - 159
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 160 - 175
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 176 - 191
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 192 - 207
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 208 - 223
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 224 - 239
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 240 - 255
	}
)

func decodeRevision(revision []byte) RevisionUInt64 {
	var t RevisionUInt64

	for _, s := range revision {
		t = t*64 + RevisionUInt64(revisionDecodingTable[s])
	}

	return t
}

func encodeRevision(revision RevisionUInt64) []byte {
	if revision == 0 {
		return []byte{}
	}

	var result [12]byte
	index := cap(result)

	for revision > 0 {
		index--
		result[index] = revisionEncodingTable[uint8(revision&0x3f)]
		revision >>= 6
	}

	return result[index:]
}

// UnmarshalJSON parses string revision document into RevisionUInt64 number
func (n *RevisionUInt64) UnmarshalJSON(revision []byte) (err error) {
	length := len(revision)

	if length > 2 {
		*n = decodeRevision(revision[1 : length-1])
	} else {
		// it can be only empty json string ""
		*n = 0
	}

	return nil
}

// MarshalJSON converts RevisionUInt64 into string revision
func (n *RevisionUInt64) MarshalJSON() ([]byte, error) {
	if *n == 0 {
		return []byte{'"', '"'}, nil // return an empty string
	}

	value := make([]byte, 0, 16)
	r := encodeRevision(*n)
	value = append(value, '"')
	value = append(value, r...)
	value = append(value, '"')
	return value, nil
}

// UnmarshalVPack parses string revision document into RevisionUInt64 number
func (n *RevisionUInt64) UnmarshalVPack(slice velocypack.Slice) error {
	source, err := slice.GetString()
	if err != nil {
		return err
	}

	*n = decodeRevision([]byte(source))
	return nil
}

// MarshalVPack converts RevisionUInt64 into string revision
func (n *RevisionUInt64) MarshalVPack() (velocypack.Slice, error) {
	var b velocypack.Builder

	value := velocypack.NewStringValue(string(encodeRevision(*n)))
	if err := b.AddValue(value); err != nil {
		return nil, err
	}

	return b.Slice()
}

// GetRevisionTree retrieves the Revision tree (Merkel tree) associated with the collection.
func (c *client) GetRevisionTree(ctx context.Context, db Database, batchId, collection string) (RevisionTree, error) {

	req, err := c.conn.NewRequest("GET", path.Join("_db", db.Name(), "_api/replication/revisions/tree"))
	if err != nil {
		return RevisionTree{}, WithStack(err)
	}

	req = req.SetQuery("batchId", batchId)
	req = req.SetQuery("collection", collection)

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return RevisionTree{}, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return RevisionTree{}, WithStack(err)
	}

	var tree RevisionTree
	if err := resp.ParseBody("", &tree); err != nil {
		return RevisionTree{}, WithStack(err)
	}

	return tree, nil
}

// GetRevisionsByRanges retrieves the revision IDs of documents within requested ranges.
func (c *client) GetRevisionsByRanges(ctx context.Context, db Database, batchId, collection string,
	minMaxRevision []RevisionMinMax, resume RevisionUInt64) (RevisionRanges, error) {

	req, err := c.conn.NewRequest("PUT", path.Join("_db", db.Name(), "_api/replication/revisions/ranges"))
	if err != nil {
		return RevisionRanges{}, WithStack(err)
	}

	req = req.SetQuery("batchId", batchId)
	req = req.SetQuery("collection", collection)
	if resume > 0 {
		req = req.SetQuery("resume", string(encodeRevision(resume)))
	}

	req, err = req.SetBodyArray(minMaxRevision, nil)
	if err != nil {
		return RevisionRanges{}, WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return RevisionRanges{}, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return RevisionRanges{}, WithStack(err)
	}

	var ranges RevisionRanges
	if err := resp.ParseBody("", &ranges); err != nil {
		return RevisionRanges{}, WithStack(err)
	}

	return ranges, nil
}

// GetRevisionDocuments retrieves documents by revision.
func (c *client) GetRevisionDocuments(ctx context.Context, db Database, batchId, collection string,
	revisions Revisions) ([]map[string]interface{}, error) {

	req, err := c.conn.NewRequest("PUT", path.Join("_db", db.Name(), "_api/replication/revisions/documents"))
	if err != nil {
		return nil, WithStack(err)
	}

	req = req.SetQuery("batchId", batchId)
	req = req.SetQuery("collection", collection)

	req, err = req.SetBody(revisions)
	if err != nil {
		return nil, WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}

	arrayResponse, err := resp.ParseArrayBody()
	if err != nil {
		return nil, WithStack(err)
	}

	documents := make([]map[string]interface{}, 0, len(arrayResponse))
	for _, a := range arrayResponse {
		document := map[string]interface{}{}
		if err = a.ParseBody("", &document); err != nil {
			return nil, WithStack(err)
		}
		documents = append(documents, document)
	}

	return documents, nil
}
