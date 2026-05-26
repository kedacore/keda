package azkustodata

import (
	"context"
	"github.com/Azure/azure-kusto-go/azkustodata/kql"
	"github.com/Azure/azure-kusto-go/azkustodata/query"
	v1 "github.com/Azure/azure-kusto-go/azkustodata/query/v1"
	queryv2 "github.com/Azure/azure-kusto-go/azkustodata/query/v2"
	"github.com/Azure/azure-kusto-go/azkustodata/value"
	"io"
	"net/http"
	"time"

	"github.com/Azure/azure-kusto-go/azkustodata/errors"
)

type Statement = *kql.Builder

// queryer provides for getting a stream of Kusto frames. Exists to allow fake Kusto streams in tests.
type queryer interface {
	io.Closer
	rawQuery(ctx context.Context, callType callType, db string, query Statement, options *queryOptions) (io.ReadCloser, error)
}

// Authorization provides the TokenProvider needed to acquire the auth token.
type Authorization struct {
	// Token provider that can be used to get the access token.
	TokenProvider *TokenProvider
}

const (
	defaultMgmtTimeout  = time.Hour
	defaultQueryTimeout = 4 * time.Minute
	clientServerDelta   = 30 * time.Second
)

// Client is a client to a Kusto instance.
type Client struct {
	conn          queryer
	endpoint      string
	auth          Authorization
	http          *http.Client
	clientDetails *ClientDetails
}

// Option is an optional argument type for New().
type Option func(c *Client)

// New returns a new Client.
func New(kcsb *ConnectionStringBuilder, options ...Option) (*Client, error) {
	tkp, err := kcsb.newTokenProvider()
	if err != nil {
		return nil, err
	}
	auth := &Authorization{
		TokenProvider: tkp,
	}
	endpoint := kcsb.DataSource

	client := &Client{auth: *auth, endpoint: endpoint, clientDetails: NewClientDetails(kcsb.ApplicationForTracing, kcsb.UserForTracing)}
	for _, o := range options {
		o(client)
	}

	if client.http == nil {
		client.http = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	conn, err := NewConn(endpoint, *auth, client.http, client.clientDetails)
	if err != nil {
		return nil, err
	}
	client.conn = conn

	return client, nil
}

func WithHttpClient(client *http.Client) Option {
	return func(c *Client) {
		c.http = client
	}
}

// QueryOption is an option type for a call to Query().
type QueryOption func(q *queryOptions) error

// Auth returns the Authorization passed to New().
func (c *Client) Auth() Authorization {
	return c.auth
}

// Endpoint returns the endpoint passed to New().
func (c *Client) Endpoint() string {
	return c.endpoint
}

type callType int8

const (
	queryCall = 1
	mgmtCall  = 2
)

func (c *Client) Mgmt(ctx context.Context, db string, kqlQuery Statement, options ...QueryOption) (v1.Dataset, error) {
	ctx, cancel := contextSetup(ctx)

	opQuery := errors.OpMgmt
	call := mgmtCall
	opts, err := setQueryOptions(ctx, opQuery, kqlQuery, call, options...)
	if err != nil {
		return nil, err
	}

	conn, err := c.getConn(callType(call), connOptions{queryOptions: opts})
	if err != nil {
		return nil, err
	}

	res, err := conn.rawQuery(ctx, callType(call), db, kqlQuery, opts)

	if err != nil {
		cancel()
		return nil, err
	}

	return v1.NewDatasetFromReader(ctx, opQuery, res)
}

func (c *Client) Query(ctx context.Context, db string, kqlQuery Statement, options ...QueryOption) (query.Dataset, error) {
	ds, err := c.IterativeQuery(ctx, db, kqlQuery, options...)
	if err != nil {
		return nil, err
	}

	return ds.ToDataset()
}

func (c *Client) IterativeQuery(ctx context.Context, db string, kqlQuery Statement, options ...QueryOption) (query.IterativeDataset, error) {
	options = append(options, V2NewlinesBetweenFrames())
	options = append(options, V2FragmentPrimaryTables())
	options = append(options, ResultsErrorReportingPlacement(ResultsErrorReportingPlacementEndOfTable))

	opts, res, err := c.rawV2(ctx, db, kqlQuery, options)
	if err != nil {
		return nil, err
	}

	frameCapacity := queryv2.DefaultIoCapacity
	if opts.v2IoCapacity != -1 {
		frameCapacity = opts.v2IoCapacity
	}

	rowCapacity := queryv2.DefaultRowCapacity
	if opts.v2RowCapacity != -1 {
		rowCapacity = opts.v2RowCapacity
	}

	fragmentCapacity := queryv2.DefaultTableCapacity
	if opts.v2TableCapacity != -1 {
		fragmentCapacity = opts.v2TableCapacity
	}

	return queryv2.NewIterativeDataset(ctx, res, frameCapacity, rowCapacity, fragmentCapacity)
}

func (c *Client) RawV2(ctx context.Context, db string, kqlQuery Statement, options []QueryOption) (io.ReadCloser, error) {

	_, res, err := c.rawV2(ctx, db, kqlQuery, options)

	return res, err

}

func (c *Client) rawV2(ctx context.Context, db string, kqlQuery Statement, options []QueryOption) (*queryOptions, io.ReadCloser, error) {
	ctx, cancel := contextSetup(ctx)
	opQuery := errors.OpQuery
	opts, err := setQueryOptions(ctx, opQuery, kqlQuery, queryCall, options...)
	if err != nil {
		return nil, nil, err
	}

	conn, err := c.getConn(queryCall, connOptions{queryOptions: opts})
	if err != nil {
		return nil, nil, err
	}

	res, err := conn.rawQuery(ctx, queryCall, db, kqlQuery, opts)

	if err != nil {
		cancel()
		return nil, nil, err
	}
	return opts, res, nil
}

func (c *Client) QueryToJson(ctx context.Context, db string, query Statement, options ...QueryOption) (string, error) {
	_, res, err := c.rawV2(ctx, db, query, options)
	if err != nil {
		return "", err
	}

	all, err := io.ReadAll(res)
	if err != nil {
		return "", err
	}

	return string(all), nil
}

func setQueryOptions(ctx context.Context, op errors.Op, query Statement, queryType int, options ...QueryOption) (*queryOptions, error) {
	opt := &queryOptions{
		requestProperties: &requestProperties{
			Options: map[string]interface{}{},
		},
		v2IoCapacity:    -1,
		v2RowCapacity:   -1,
		v2TableCapacity: -1,
	}

	for _, o := range options {
		if err := o(opt); err != nil {
			return nil, errors.ES(op, errors.KClientArgs, "QueryValues in the the Stmt were incorrect: %s", err).SetNoRetry()
		}
	}

	CalculateTimeout(ctx, opt, queryType)

	if query.SupportsInlineParameters() {
		if opt.requestProperties.QueryParameters.Count() != 0 {
			return nil, errors.ES(op, errors.KClientArgs, "kusto.Stmt does not support the QueryParameters option. Construct your query using `kql.New`").SetNoRetry()
		}
		params, err := query.GetParameters()
		if err != nil {
			return nil, errors.ES(op, errors.KClientArgs, "Parameter validation error: %s", err).SetNoRetry()
		}

		opt.requestProperties.Parameters = params
	}
	return opt, nil
}

var nower = time.Now

func CalculateTimeout(ctx context.Context, opt *queryOptions, queryType int) {
	// If the user has specified a timeout, use that.
	if val, ok := opt.requestProperties.Options[NoRequestTimeoutValue]; ok && val.(bool) {
		return
	}
	if _, ok := opt.requestProperties.Options[ServerTimeoutValue]; ok {
		return
	}

	// Otherwise use the context deadline, if it exists. If it doesn't, use the default timeout.
	if deadline, ok := ctx.Deadline(); ok {
		opt.requestProperties.Options[ServerTimeoutValue] = value.TimespanString(deadline.Sub(nower()))
		return
	}

	var timeout time.Duration
	switch queryType {
	case queryCall:
		timeout = defaultQueryTimeout
	case mgmtCall:
		timeout = defaultMgmtTimeout
	}
	opt.requestProperties.Options[ServerTimeoutValue] = value.TimespanString(timeout)
}

func (c *Client) getConn(callType callType, options connOptions) (queryer, error) {
	switch callType {
	case queryCall:
		return c.conn, nil
	case mgmtCall:
		delete(options.queryOptions.requestProperties.Options, "results_progressive_enabled")
		return c.conn, nil
	default:
		return nil, errors.ES(errors.OpServConn, errors.KInternal, "an unknown calltype was passed to getConn()")
	}
}

func contextSetup(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func (c *Client) HttpClient() *http.Client {
	return c.http
}

func (c *Client) ClientDetails() *ClientDetails {
	return c.clientDetails
}

func (c *Client) Close() error {
	var err error
	if c.conn != nil {
		err = c.conn.Close()
	}
	return err
}
