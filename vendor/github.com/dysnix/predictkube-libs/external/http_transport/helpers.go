package http_transport

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	libs "github.com/dysnix/predictkube-libs/external/configs"
	"github.com/dysnix/predictkube-libs/external/enums"
)

func RequestJob(ctx context.Context, remoteClient *http.Client, logger *zap.SugaredLogger, backendURL, method string, requestBody string, requestHeaders map[string]string) (respStatus int, compressed bool, dataClear *bytes.Buffer, err error) {
	var (
		req  *http.Request
		resp *http.Response
	)

	req, err = http.NewRequestWithContext(ctx, method, backendURL, io.NopCloser(strings.NewReader(requestBody)))
	if err != nil {
		return http.StatusInternalServerError, false, nil, err
	}

	var buf bytes.Buffer

	defer func() {
		buf.Reset()
	}()

	var useCompression bool
	for headerKey, headerValue := range requestHeaders {
		if /*(headerKey == "Accept-Encoding" || headerKey == "Content-Encoding")*/ headerKey == "Content-Encoding" && headerValue == "gzip" && len(requestBody) > 0 {
			useCompression = true
			g := gzip.NewWriter(&buf)
			if _, err = g.Write([]byte(requestBody)); err != nil {
				return http.StatusInternalServerError, false, nil, err
			}

			_ = g.Close()
			req.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
		} else {
			req.Body = io.NopCloser(strings.NewReader(requestBody))
		}

		req.Header.Set(headerKey, headerValue)
	}

	resp, err = remoteClient.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()

	if err != nil {
		return resp.StatusCode, false, nil, err
	}

	if resp != nil && resp.Body != nil {
		var (
			dataOrig, dataClear bytes.Buffer
		)

		defer func() {
			dataOrig.Reset()
		}()

		if _, err = io.Copy(&dataOrig, resp.Body); err != nil {
			logger.Errorf("clone response body error: %3v", err)
			return http.StatusInternalServerError, false, nil, err
		}

		if useCompression {
			compressed = true
			reader, _ := gzip.NewReader(bytes.NewReader(dataOrig.Bytes()))
			if _, err = io.Copy(&dataClear, reader); err != nil {
				logger.Errorf("clone gzipped response body error: %3v", err)
				return http.StatusInternalServerError, false, nil, err
			}
		} else {
			if _, err = io.Copy(&dataClear, bytes.NewReader(dataOrig.Bytes())); err != nil {
				logger.Errorf("clone origin response body error: %3v", err)
				return http.StatusInternalServerError, false, nil, err
			}
		}

		return resp.StatusCode, compressed, &dataClear, nil
	}

	return http.StatusInternalServerError, false, nil, fmt.Errorf("response or response body is empty")
}

func InitHTTPClient(t enums.TransportType, conf *libs.HTTPTransport) (httpClient *http.Client, transportStats HttpTransportWithRequestStats, err error) {
	var roundTripper http.RoundTripper

	switch t {
	case enums.NetHTTP:
		customTransport, err := NewNetHttpTransport(
			libs.SetTransportConfigs(conf),
			libs.SetTLS(&tls.Config{InsecureSkipVerify: false}),
		)

		if err != nil {
			return nil, nil, err
		}

		roundTripper = customTransport
		transportStats = customTransport
	case enums.FastHTTP:
		if roundTripper, err = NewHttpTransport(
			libs.SetTransportConfigs(&libs.HTTPTransport{
				MaxIdleConnDuration: 10,
				ReadTimeout:         time.Second * 15,
				WriteTimeout:        time.Second * 15,
			}),
			libs.SetTLS(&tls.Config{InsecureSkipVerify: false}),
		); err != nil {
			return nil, nil, err
		}
	}

	return &http.Client{
		Transport: roundTripper,
		Timeout:   time.Second * 15,
	}, transportStats, nil
}
