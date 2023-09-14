package kusto

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/google/uuid"
)

type DataFormatForStreaming interface {
	CamelCase() string
	KnownOrDefault() DataFormatForStreaming
}

var (
	streamingIngestDefaultTimeout = 10 * time.Minute
)

func (c *Conn) StreamIngest(ctx context.Context, db, table string, payload io.Reader, format DataFormatForStreaming, mappingName string, clientRequestId string, isBlobUri bool) error {
	streamUrl, err := url.Parse(c.endStreamIngest.String())
	if err != nil {
		return errors.ES(errors.OpIngestStream, errors.KClientArgs, "could not parse the stream endpoint(%s): %s", c.endStreamIngest.String(), err).SetNoRetry()
	}
	path, err := url.JoinPath(streamUrl.Path, db, table)
	if err != nil {
		return errors.ES(errors.OpIngestStream, errors.KClientArgs, "could not join the stream endpoint(%s) with the db(%s) and table(%s): %s", c.endStreamIngest.String(), db, table, err).SetNoRetry()
	}
	streamUrl.Path = path

	qv := url.Values{}
	if mappingName != "" {
		qv.Add("mappingName", mappingName)
	}
	qv.Add("streamFormat", format.KnownOrDefault().CamelCase())
	if isBlobUri {
		qv.Add("sourceKind", "uri")
	}
	streamUrl.RawQuery = qv.Encode()

	var closeablePayload io.ReadCloser
	var ok bool
	if closeablePayload, ok = payload.(io.ReadCloser); !ok {
		closeablePayload = io.NopCloser(payload)
	}

	if clientRequestId == "" {
		clientRequestId = "KGC.executeStreaming;" + uuid.New().String()
	}

	properties := requestProperties{}
	properties.ClientRequestID = clientRequestId
	headers := c.getHeaders(properties)
	headers.Del("Content-Type")
	if !isBlobUri {
		headers.Add("Content-Encoding", "gzip")
	}

	if _, ok := ctx.Deadline(); !ok {
		ctx, _ = context.WithTimeout(ctx, streamingIngestDefaultTimeout)
	}

	_, body, err := c.doRequestImpl(ctx, errors.OpIngestStream, streamUrl, closeablePayload, headers, fmt.Sprintf("With db: %s, table: %s, mappingName: %s, clientRequestId: %s", db, table, mappingName, clientRequestId))
	if body != nil {
		body.Close()
	}

	if err != nil {
		return errors.ES(errors.OpIngestStream, errors.KHTTPError, "streaming ingestion failed: endpoint(%s): %s", streamUrl.String(), err)
	}

	return nil
}
