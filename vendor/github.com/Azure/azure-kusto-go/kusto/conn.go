package kusto

// conn.go holds the connection to the Kusto server and provides methods to do queries
// and receive Kusto frames back.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
	v1 "github.com/Azure/azure-kusto-go/kusto/internal/frames/v1"
	v2 "github.com/Azure/azure-kusto-go/kusto/internal/frames/v2"
	"github.com/Azure/azure-kusto-go/kusto/internal/response"
	"github.com/Azure/azure-kusto-go/kusto/internal/version"

	"github.com/Azure/go-autorest/autorest"
	"github.com/google/uuid"
)

var validURL = regexp.MustCompile(`https://([a-zA-Z0-9_-]+\.){1,2}.*`)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// conn provides connectivity to a Kusto instance.
type conn struct {
	auth                           autorest.Authorizer
	endMgmt, endQuery, streamQuery *url.URL
	client                         *http.Client
}

// newConn returns a new conn object with an injected http.Client
func newConn(endpoint string, auth Authorization, client *http.Client) (*conn, error) {
	if !validURL.MatchString(endpoint) {
		return nil, errors.ES(errors.OpServConn, errors.KClientArgs, "endpoint is not valid(%s), should be https://<cluster name>.*", endpoint).SetNoRetry()
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.ES(errors.OpServConn, errors.KClientArgs, "could not parse the endpoint(%s): %s", endpoint, err).SetNoRetry()
	}

	c := &conn{
		auth:        auth.Authorizer,
		endMgmt:     &url.URL{Scheme: "https", Host: u.Host, Path: "/v1/rest/mgmt"},
		endQuery:    &url.URL{Scheme: "https", Host: u.Host, Path: "/v2/rest/query"},
		streamQuery: &url.URL{Scheme: "https", Host: u.Host, Path: "/v1/rest/ingest/"},
		client:      client,
	}

	return c, nil
}

type queryMsg struct {
	DB         string            `json:"db"`
	CSL        string            `json:"csl"`
	Properties requestProperties `json:"properties,omitempty"`
}

type connOptions struct {
	queryOptions *queryOptions
	mgmtOptions  *mgmtOptions
}

// query makes a query for the purpose of extracting data from Kusto. Context can be used to set
// a timeout or cancel the query. Queries cannot take longer than 5 minutes.
func (c *conn) query(ctx context.Context, db string, query Stmt, options *queryOptions) (execResp, error) {
	if strings.HasPrefix(strings.TrimSpace(query.String()), ".") {
		return execResp{}, errors.ES(errors.OpQuery, errors.KClientArgs, "a Stmt to Query() cannot begin with a period(.), only Mgmt() calls can do that").SetNoRetry()
	}

	return c.execute(ctx, execQuery, db, query, *options.requestProperties)
}

// mgmt is used to do management queries to Kusto.
func (c *conn) mgmt(ctx context.Context, db string, query Stmt, options *mgmtOptions) (execResp, error) {
	return c.execute(ctx, execMgmt, db, query, *options.requestProperties)
}

func (c *conn) queryToJson(ctx context.Context, db string, query Stmt, options *queryOptions) (string, error) {
	_, _, _, body, e := c.doRequest(ctx, execQuery, db, query, *options.requestProperties)
	if e != nil {
		return "", e
	}

	all, e := io.ReadAll(body)
	return string(all), e
}

const (
	execQuery = 1
	execMgmt  = 2
)

type execResp struct {
	reqHeader  http.Header
	respHeader http.Header
	frameCh    chan frames.Frame
}

func (c *conn) execute(ctx context.Context, execType int, db string, query Stmt, properties requestProperties) (execResp, error) {
	op, header, resp, body, e := c.doRequest(ctx, execType, db, query, properties)
	if e != nil {
		return execResp{}, e
	}

	var dec frames.Decoder
	switch execType {
	case execMgmt:
		dec = &v1.Decoder{}
	case execQuery:
		dec = &v2.Decoder{}
	default:
		return execResp{}, errors.ES(op, errors.KInternal, "unknown execution type was %v", execType).SetNoRetry()
	}

	frameCh := dec.Decode(ctx, body, op)

	return execResp{reqHeader: header, respHeader: resp.Header, frameCh: frameCh}, nil
}

func (c *conn) doRequest(ctx context.Context, execType int, db string, query Stmt, properties requestProperties) (errors.Op, http.Header, *http.Response, io.ReadCloser, error) {
	var op errors.Op
	if execType == execQuery {
		op = errors.OpQuery
	} else if execType == execMgmt {
		op = errors.OpMgmt
	}

	header := http.Header{}
	header.Add("Accept", "application/json")
	header.Add("Accept-Encoding", "gzip")
	header.Add("x-ms-client-version", "Kusto.Go.Client: "+version.Kusto)
	header.Add("Content-Type", "application/json; charset=utf-8")
	header.Add("x-ms-client-request-id", "KGC.execute;"+uuid.New().String())

	if properties.Application != "" {
		header.Add("x-ms-app", properties.Application)
	}

	if properties.User != "" {
		header.Add("x-ms-user", properties.User)
	}

	var endpoint *url.URL
	buff := bufferPool.Get().(*bytes.Buffer)
	buff.Reset()
	defer bufferPool.Put(buff)

	switch execType {
	case execQuery, execMgmt:
		var err error
		err = json.NewEncoder(buff).Encode(
			queryMsg{
				DB:         db,
				CSL:        query.String(),
				Properties: properties,
			},
		)
		if err != nil {
			return 0, nil, nil, nil, errors.E(op, errors.KInternal, fmt.Errorf("could not JSON marshal the Query message: %w", err))
		}
		if execType == execQuery {
			endpoint = c.endQuery
		} else {
			endpoint = c.endMgmt
		}
	default:
		return 0, nil, nil, nil, errors.ES(op, errors.KInternal, "internal error: did not understand the type of execType: %d", execType)
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    endpoint,
		Header: header,
		Body:   io.NopCloser(buff),
	}

	var err error
	prep := c.auth.WithAuthorization()
	req, err = prep(autorest.CreatePreparer()).Prepare(req)
	if err != nil {
		return 0, nil, nil, nil, errors.E(op, errors.KInternal, err)
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		// TODO(jdoak): We need a http error unwrap function that pulls out an *errors.Error.
		return 0, nil, nil, nil, errors.E(op, errors.KHTTPError, fmt.Errorf("with query %q: %w", query.String(), err))
	}

	body, err := response.TranslateBody(resp, op)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, nil, nil, nil, errors.HTTP(op, resp.Status, resp.StatusCode, body, fmt.Sprintf("error from Kusto endpoint for query %q: ", query.String()))
	}
	return op, header, resp, body, nil
}

func (c *conn) Close() error {
	if c.auth == nil {
		return nil
	}

	if closer, ok := c.auth.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
