package couchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/go-kivik/couchdb/v3/chttp"
	"github.com/go-kivik/kivik/v3/driver"
)

const (
	pathIndex = "_index"
)

func (d *db) CreateIndex(ctx context.Context, ddoc, name string, index interface{}, opts map[string]interface{}) error {
	reqPath := pathIndex
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	indexObj, err := deJSONify(index)
	if err != nil {
		return err
	}
	parameters := struct {
		Index interface{} `json:"index"`
		Ddoc  string      `json:"ddoc,omitempty"`
		Name  string      `json:"name,omitempty"`
	}{
		Index: indexObj,
		Ddoc:  ddoc,
		Name:  name,
	}
	options := &chttp.Options{
		Body: chttp.EncodeBody(parameters),
	}
	_, err = d.Client.DoError(ctx, http.MethodPost, d.path(reqPath), options)
	return err
}

func (d *db) GetIndexes(ctx context.Context, opts map[string]interface{}) ([]driver.Index, error) {
	reqPath := pathIndex
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	var result struct {
		Indexes []driver.Index `json:"indexes"`
	}
	_, err := d.Client.DoJSON(ctx, http.MethodGet, d.path(reqPath), nil, &result)
	return result.Indexes, err
}

func (d *db) DeleteIndex(ctx context.Context, ddoc, name string, opts map[string]interface{}) error {
	if ddoc == "" {
		return missingArg("ddoc")
	}
	if name == "" {
		return missingArg("name")
	}
	reqPath := pathIndex
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	path := fmt.Sprintf("%s/%s/json/%s", reqPath, ddoc, name)
	_, err := d.Client.DoError(ctx, http.MethodDelete, d.path(path), nil)
	return err
}

func (d *db) Find(ctx context.Context, query interface{}, opts map[string]interface{}) (driver.Rows, error) {
	reqPath := "_find"
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	options := &chttp.Options{
		GetBody: chttp.BodyEncoder(query),
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	resp, err := d.Client.DoReq(ctx, http.MethodPost, d.path(reqPath), options)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newFindRows(ctx, resp.Body), nil
}

type queryPlan struct {
	DBName   string                 `json:"dbname"`
	Index    map[string]interface{} `json:"index"`
	Selector map[string]interface{} `json:"selector"`
	Options  map[string]interface{} `json:"opts"`
	Limit    int64                  `json:"limit"`
	Skip     int64                  `json:"skip"`
	Fields   fields                 `json:"fields"`
	Range    map[string]interface{} `json:"range"`
}

type fields []interface{}

func (f *fields) UnmarshalJSON(data []byte) error {
	if string(data) == `"all_fields"` {
		return nil
	}
	var i []interface{}
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	newFields := make([]interface{}, len(i))
	copy(newFields, i)
	*f = newFields
	return nil
}

func (d *db) Explain(ctx context.Context, query interface{}, opts map[string]interface{}) (*driver.QueryPlan, error) {
	reqPath := "_explain"
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	options := &chttp.Options{
		GetBody: chttp.BodyEncoder(query),
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	var plan queryPlan
	if _, err := d.Client.DoJSON(ctx, http.MethodPost, d.path(reqPath), options, &plan); err != nil {
		return nil, err
	}
	return &driver.QueryPlan{
		DBName:   plan.DBName,
		Index:    plan.Index,
		Selector: plan.Selector,
		Options:  plan.Options,
		Limit:    plan.Limit,
		Skip:     plan.Skip,
		Fields:   plan.Fields,
		Range:    plan.Range,
	}, nil
}
