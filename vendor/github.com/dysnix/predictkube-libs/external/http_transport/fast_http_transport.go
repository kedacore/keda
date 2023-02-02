package http_transport

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
	tr "k8s.io/client-go/transport"

	"github.com/dysnix/predictkube-libs/external/configs"
)

// Transport implements the http.RoundTripper interface with
// the github.com/valyala/fasthttp HTTP client.
type transport struct {
	client  *fasthttp.Client
	conf    configs.TransportGetter
	tlsConf *tls.Config
}

type FastHttpTransport interface {
	http.RoundTripper
	configs.SignalCloser
}

func NewHttpTransport(options ...configs.TransportOption) (out *transport, err error) {
	out = &transport{client: &fasthttp.Client{}}

	for _, op := range options {
		err := op(out)
		if err != nil {
			return nil, err
		}
	}

	if out.conf != nil {
		out.client.MaxIdleConnDuration = out.conf.GetTransportConfigs().MaxIdleConnDuration
		out.client.ReadTimeout = out.conf.GetTransportConfigs().ReadTimeout
		out.client.WriteTimeout = out.conf.GetTransportConfigs().WriteTimeout
	}

	if out.tlsConf == nil {
		out.client.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		out.client.TLSConfig = out.tlsConf
	}

	return out, nil
}

func (s *transport) SetConfigs(configs configs.TransportGetter) {
	s.conf = configs
}

func (s *transport) SetTLS(conf *tls.Config) {
	s.tlsConf = conf
}

func NewTransport(opts *configs.HTTPTransport, tlsConf ...tr.TLSConfig) *transport {
	var tlsC *tls.Config
	if len(tlsConf) > 0 {
		if tlsConf[0].GetCertHolder.GetCert != nil {
			if certs, err := tlsConf[0].GetCertHolder.GetCert(); err == nil && certs != nil {
				tlsC = &tls.Config{
					Certificates: []tls.Certificate{*certs},
				}
			}
		}
	} else {
		tlsC = &tls.Config{InsecureSkipVerify: true}
	}

	return &transport{
		client: &fasthttp.Client{
			MaxIdleConnDuration: opts.MaxIdleConnDuration,
			ReadTimeout:         opts.ReadTimeout,
			WriteTimeout:        opts.WriteTimeout,
			// nolint:gosec
			TLSConfig: tlsC,
		},
	}
}

// RoundTrip performs the request and returns a response or error
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	freq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(freq)

	fres := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fres)

	copyRequest(freq, req)

	err := t.client.Do(freq, fres)
	if err != nil {
		return nil, err
	}

	res := &http.Response{Header: make(http.Header)}
	copyResponse(res, fres)

	return res, nil
}

func (t *transport) Close() {
	t.client.CloseIdleConnections()
}

// copyRequest converts a http.Request to fasthttp.Request
func copyRequest(dst *fasthttp.Request, src *http.Request) {
	if src.Method == fasthttp.MethodGet && src.Body != nil {
		src.Method = fasthttp.MethodPost
	}

	dst.SetHost(src.Host)
	dst.SetRequestURI(src.URL.String())

	dst.Header.SetRequestURI(src.URL.String())
	dst.Header.SetMethod(src.Method)

	for k, vv := range src.Header {
		for _, v := range vv {
			dst.Header.Set(k, v)
		}
	}

	if src.Body != nil {
		dst.SetBodyStream(bodyCloserReader{
			body: src.Body,
		}, -1)
	}
}

// copyResponse converts a http.Response to fasthttp.Response
func copyResponse(dst *http.Response, src *fasthttp.Response) {
	dst.StatusCode = src.StatusCode()

	src.Header.VisitAll(func(k, v []byte) {
		dst.Header.Set(string(k), string(v))
	})

	// Cast to a string to make a copy seeing as src.Body() won't
	// be valid after the response is released back to the pool (fasthttp.ReleaseResponse).
	dst.Body = ioutil.NopCloser(strings.NewReader(string(src.Body())))
}

type bodyCloserReader struct {
	body io.ReadCloser
}

func (bcr bodyCloserReader) Read(p []byte) (n int, err error) {
	n, err = bcr.body.Read(p)

	if err != nil {
		_ = bcr.body.Close()
	}

	return n, err
}
