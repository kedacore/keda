package http_transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dysnix/predictkube-libs/external/configs"
)

const (
	StatsKey = "transportStatsKey"
)

// Transport implements the http.RoundTripper interface with
// the net/http client.
type netHttpTransport struct {
	rtp        *http.Transport
	dialer     *net.Dialer
	statsStore sync.Map
	conf       configs.TransportGetter
	tlsConf    *tls.Config
}

type HttpTransport interface {
	http.RoundTripper
	configs.SignalCloser
}

func NewNetHttpTransport(options ...configs.TransportOption) (out *netHttpTransport, err error) {
	out = &netHttpTransport{}

	for _, op := range options {
		err := op(out)
		if err != nil {
			return nil, err
		}
	}

	out.dialer = &net.Dialer{
		Timeout:   out.conf.GetTransportConfigs().NetTransport.DialTimeout,
		KeepAlive: out.conf.GetTransportConfigs().NetTransport.KeepAlive,
	}

	tmpTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if out.conf != nil {
		tmpTransport.DialContext = out.dial
		tmpTransport.DisableKeepAlives = out.conf.GetTransportConfigs().NetTransport.DisableKeepAlives
		tmpTransport.DisableCompression = out.conf.GetTransportConfigs().NetTransport.DisableCompression
		tmpTransport.TLSHandshakeTimeout = out.conf.GetTransportConfigs().NetTransport.TLSHandshakeTimeout
		tmpTransport.MaxIdleConns = out.conf.GetTransportConfigs().NetTransport.MaxIdleConns
		tmpTransport.MaxIdleConnsPerHost = out.conf.GetTransportConfigs().NetTransport.MaxIdleConnsPerHost
		tmpTransport.MaxConnsPerHost = out.conf.GetTransportConfigs().NetTransport.MaxConnsPerHost
		tmpTransport.IdleConnTimeout = out.conf.GetTransportConfigs().MaxIdleConnDuration
		tmpTransport.ResponseHeaderTimeout = out.conf.GetTransportConfigs().NetTransport.ResponseHeaderTimeout
		tmpTransport.ExpectContinueTimeout = out.conf.GetTransportConfigs().NetTransport.ExpectContinueTimeout
		tmpTransport.MaxResponseHeaderBytes = out.conf.GetTransportConfigs().NetTransport.MaxResponseHeaderBytes

		if out.conf.GetTransportConfigs().NetTransport.Buffer != nil {
			tmpTransport.WriteBufferSize = int(out.conf.GetTransportConfigs().NetTransport.Buffer.WriteBufferSize)
			tmpTransport.ReadBufferSize = int(out.conf.GetTransportConfigs().NetTransport.Buffer.ReadBufferSize)
		}
	}

	if out.tlsConf == nil {
		tmpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		tmpTransport.TLSClientConfig = out.tlsConf
	}

	out.rtp = tmpTransport

	return out, nil
}

func (t *netHttpTransport) Close() {
	t.rtp.CloseIdleConnections()
	fmt.Println("Close netHttpTransport")
}

func (t *netHttpTransport) SetConfigs(configs configs.TransportGetter) {
	t.conf = configs
}

func (t *netHttpTransport) SetTLS(conf *tls.Config) {
	t.tlsConf = conf
}

func (t *netHttpTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var (
		event requestStats
		k     string
	)

	event.reqStart = time.Now()

	if key := GetFromContext(r.Context(), StatsKey); key != nil {
		var ok bool
		if k, ok = key.(string); ok && len(k) > 0 {
			if old, ok := t.statsStore.Load(k); ok {
				event.connStart = old.(requestStats).connStart
				event.connEnd = old.(requestStats).connEnd
			}

		}
	}

	resp, err := t.rtp.RoundTrip(r)
	event.reqEnd = time.Now()

	t.statsStore.Store(k, event)

	return resp, err
}

func (t *netHttpTransport) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	var (
		event requestStats
		k     string
	)

	if key := GetFromContext(ctx, StatsKey); key != nil {
		var ok bool
		if k, ok = key.(string); ok && len(k) > 0 {
			if old, ok := t.statsStore.Load(k); ok {
				event.reqStart = old.(requestStats).reqStart
				event.reqEnd = old.(requestStats).reqEnd
			}
		}
	}

	event.connStart = time.Now()
	cn, err := t.dialer.Dial(network, addr)
	event.connEnd = time.Now()

	t.statsStore.Store(k, event)

	return cn, err
}

type HttpTransportWithRequestStats interface {
	ReqDuration(context.Context) time.Duration
	ConnDuration(context.Context) time.Duration
	ReqLifetime(context.Context) time.Duration
}

func (t *netHttpTransport) ReqDuration(ctx context.Context) (d time.Duration) {
	if key := GetFromContext(ctx, StatsKey); key != nil {
		if k, ok := key.(string); ok && len(k) > 0 {
			if old, ok := t.statsStore.Load(k); ok {
				if rs, ok := old.(requestStats); ok {
					d = rs.reqEnd.Sub(rs.reqStart) - rs.connEnd.Sub(rs.connStart)
				}
			}
		}
	}

	return d
}

func (t *netHttpTransport) ConnDuration(ctx context.Context) (d time.Duration) {
	if key := GetFromContext(ctx, StatsKey); key != nil {
		if k, ok := key.(string); ok && len(k) > 0 {
			if old, ok := t.statsStore.Load(k); ok {
				if rs, ok := old.(requestStats); ok {
					d = rs.connEnd.Sub(rs.connStart)
				}
			}
		}
	}

	return d
}

func (t *netHttpTransport) ReqLifetime(ctx context.Context) (d time.Duration) {
	if key := GetFromContext(ctx, StatsKey); key != nil {
		if k, ok := key.(string); ok && len(k) > 0 {
			if old, ok := t.statsStore.Load(k); ok {
				if rs, ok := old.(requestStats); ok {
					d = rs.reqEnd.Sub(rs.reqStart)
				}
			}
		}
	}

	return d
}
