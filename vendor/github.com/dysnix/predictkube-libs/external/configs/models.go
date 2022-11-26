package configs

import (
	"encoding/json"
	"fmt"
	"time"

	str2duration "github.com/xhit/go-str2duration/v2"

	"github.com/dysnix/predictkube-libs/external/enums"
	tc "github.com/dysnix/predictkube-libs/external/types_convertation"
)

type Base struct {
	IsDebugMode bool `yaml:"debugMode" json:"debug_mode"`
	Profiling   Profiling
	Monitoring  Monitoring
	Single      *Single `yaml:"single,omitempty" json:"single,omitempty"`
}

type Client struct {
	ClusterID string `yaml:"clusterId" json:"cluster_id" validate:"uuid"`
	Name      string
	Token     string `validate:"jwt"`
}

type Single struct {
	Enabled       bool
	Host          string `validate:"host_if_enabled"`
	Port          uint16 `validate:"port_if_enabled"`
	Name          string
	Concurrency   uint          `validate:"required_if=Enabled true,gt=1,lte=1000000"`
	Buffer        *Buffer       `yaml:"buffer,omitempty" json:"buffer,omitempty" validate:"required_if=Enabled true"`
	TCPKeepalive  *TCPKeepalive `yaml:"tcpKeepalive,omitempty" json:"tcpKeepalive,omitempty" validate:"required_if=Enabled true"`
	HTTPTransport HTTPTransport
}

type Buffer struct {
	ReadBufferSize  uint `yaml:"readBufferSize" json:"read_buffer_size" validate:"gte=4096"`
	WriteBufferSize uint `yaml:"writeBufferSize" json:"write_buffer_size" validate:"gte=4096"`
}

func (b *Buffer) MarshalJSON() ([]byte, error) {
	type alias struct {
		ReadBufferSize  string `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string `yaml:"writeBufferSize" json:"write_buffer_size"`
	}

	if b == nil {
		*b = Buffer{}
	}

	return json.Marshal(alias{
		ReadBufferSize:  tc.BytesSize(float64(b.ReadBufferSize)),
		WriteBufferSize: tc.BytesSize(float64(b.WriteBufferSize)),
	})
}

func (b *Buffer) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		ReadBufferSize  string `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string `yaml:"writeBufferSize" json:"write_buffer_size"`
	}
	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if b == nil {
		*b = Buffer{}
	}

	var tmpB int64
	if tmpB, err = tc.RAMInBytes(tmp.ReadBufferSize); err != nil {
		return err
	}

	b.ReadBufferSize = uint(tmpB)

	if tmpB, err = tc.RAMInBytes(tmp.WriteBufferSize); err != nil {
		return err
	}

	b.WriteBufferSize = uint(tmpB)

	return nil
}

func (b *Buffer) MarshalYAML() (interface{}, error) {
	type alias struct {
		ReadBufferSize  string `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string `yaml:"writeBufferSize" json:"write_buffer_size"`
	}

	if b == nil {
		*b = Buffer{}
	}

	return alias{
		ReadBufferSize:  tc.BytesSize(float64(b.ReadBufferSize)),
		WriteBufferSize: tc.BytesSize(float64(b.WriteBufferSize)),
	}, nil
}

func (b *Buffer) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		ReadBufferSize  string `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string `yaml:"writeBufferSize" json:"write_buffer_size"`
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if b == nil {
		*b = Buffer{}
	}

	var tmpB int64
	if tmpB, err = tc.RAMInBytes(tmp.ReadBufferSize); err != nil {
		return err
	}

	b.ReadBufferSize = uint(tmpB)

	if tmpB, err = tc.RAMInBytes(tmp.WriteBufferSize); err != nil {
		return err
	}

	b.WriteBufferSize = uint(tmpB)

	return nil
}

type TCPKeepalive struct {
	Enabled bool
	Period  time.Duration `validate:"required,gt=0"`
}

func (k *TCPKeepalive) MarshalJSON() ([]byte, error) {
	type alias struct {
		Enabled bool
		Period  string
	}

	if k == nil {
		*k = TCPKeepalive{}
	}

	return json.Marshal(alias{
		Enabled: k.Enabled,
		Period:  HumanDuration(k.Period),
	})
}

func (k *TCPKeepalive) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		Enabled bool
		Period  string
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if k == nil {
		*k = TCPKeepalive{}
	}

	k.Enabled = tmp.Enabled

	k.Period, err = str2duration.ParseDuration(tmp.Period)
	if err != nil {
		return err
	}

	return nil
}

func (k *TCPKeepalive) MarshalYAML() (interface{}, error) {
	type alias struct {
		Enabled bool
		Period  string
	}

	if k == nil {
		*k = TCPKeepalive{}
	}

	return alias{
		Enabled: k.Enabled,
		Period:  HumanDuration(k.Period),
	}, nil
}

func (k *TCPKeepalive) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		Enabled bool
		Period  string
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if k == nil {
		*k = TCPKeepalive{}
	}

	k.Enabled = tmp.Enabled

	k.Period, err = str2duration.ParseDuration(tmp.Period)
	if err != nil {
		return err
	}

	return nil
}

type Monitoring struct {
	Enabled bool
	Host    string `yaml:"host,omitempty" json:"host,omitempty" validate:"host_if_enabled"`
	Port    uint16 `yaml:"port,omitempty" json:"port,omitempty" validate:"port_if_enabled"`
}

type Profiling struct {
	Enabled bool
	Host    string `yaml:"host,omitempty" json:"host,omitempty" validate:"host_if_enabled"`
	Port    uint16 `yaml:"port,omitempty" json:"port,omitempty" validate:"port_if_enabled"`
}

type CronStr string

type Informer struct {
	Resource string        `yaml:"resource" json:"resource" validate:"required"`
	Interval time.Duration `yaml:"interval" json:"interval" validate:"required,gt=0"`
}

type K8sCloudWatcher struct {
	CtxPath   string     `yaml:"kubeConfigPath" json:"kube_config_path" validate:"required,file"`
	Informers []Informer `yaml:"informers" json:"informers" validate:"required,gt=0"`
}

type GRPC struct {
	Enabled       bool
	UseReflection bool        `yaml:"useReflection" json:"use_reflection"`
	Compression   Compression `yaml:"compression" json:"compression"`
	Conn          *Connection `yaml:"connection" json:"connection" validate:"required"`
	Keepalive     *Keepalive  `yaml:"keepalive" json:"keepalive"`
}

type Compression struct {
	Enabled bool                  `yaml:"enabled" json:"enabled"`
	Type    enums.CompressionType `yaml:"type" json:"type"`
}

type Connection struct {
	Host            string        `yaml:"host" json:"host" validate:"grpc_host"`
	Port            uint16        `yaml:"port" json:"port" validate:"required,gt=0"`
	ReadBufferSize  uint          `yaml:"readBufferSize" json:"read_buffer_size" validate:"required,gte=4096"`
	WriteBufferSize uint          `yaml:"writeBufferSize" json:"write_buffer_size" validate:"required,gte=4096"`
	MaxMessageSize  uint          `yaml:"maxMessageSize" json:"max_message_size" validate:"required,gte=2048"`
	Insecure        bool          `yaml:"insecure" json:"insecure"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout" validate:"gte=0"`
}

func (c *Connection) MarshalJSON() ([]byte, error) {
	type alias struct {
		Host            string  `yaml:"host" json:"host"`
		Port            uint16  `yaml:"port" json:"port"`
		ReadBufferSize  string  `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string  `yaml:"writeBufferSize" json:"write_buffer_size"`
		MaxMessageSize  string  `yaml:"maxMessageSize" json:"max_message_size"`
		Insecure        bool    `yaml:"insecure" json:"insecure"`
		Timeout         *string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	}

	if c == nil {
		*c = Connection{}
	}

	return json.Marshal(alias{
		Host:            c.Host,
		Port:            c.Port,
		ReadBufferSize:  tc.BytesSize(float64(c.ReadBufferSize)),
		WriteBufferSize: tc.BytesSize(float64(c.WriteBufferSize)),
		MaxMessageSize:  tc.BytesSize(float64(c.MaxMessageSize)),
		Insecure:        c.Insecure,
		Timeout:         tc.String(ConvertDurationToStr(c.Timeout)),
	})
}

func (c *Connection) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		Host            string  `yaml:"host" json:"host"`
		Port            uint16  `yaml:"port" json:"port"`
		ReadBufferSize  string  `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string  `yaml:"writeBufferSize" json:"write_buffer_size"`
		MaxMessageSize  string  `yaml:"maxMessageSize" json:"max_message_size"`
		Insecure        bool    `yaml:"insecure" json:"insecure"`
		Timeout         *string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	}
	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if c == nil {
		*c = Connection{}
	}

	if tmp.Timeout != nil {
		c.Timeout, err = str2duration.ParseDuration(*tmp.Timeout)
		if err != nil {
			return err
		}
	}

	c.Host = tmp.Host
	c.Port = tmp.Port

	var tmpB int64
	if tmpB, err = tc.RAMInBytes(tmp.ReadBufferSize); err != nil {
		return err
	}

	c.ReadBufferSize = uint(tmpB)

	if tmpB, err = tc.RAMInBytes(tmp.WriteBufferSize); err != nil {
		return err
	}

	c.WriteBufferSize = uint(tmpB)

	if tmpB, err = tc.RAMInBytes(tmp.MaxMessageSize); err != nil {
		return err
	}

	c.MaxMessageSize = uint(tmpB)
	c.Insecure = tmp.Insecure

	return nil
}

func (c *Connection) MarshalYAML() (interface{}, error) {
	type alias struct {
		Host            string  `yaml:"host" json:"host"`
		Port            uint16  `yaml:"port" json:"port"`
		ReadBufferSize  string  `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string  `yaml:"writeBufferSize" json:"write_buffer_size"`
		MaxMessageSize  string  `yaml:"maxMessageSize" json:"max_message_size"`
		Insecure        bool    `yaml:"insecure" json:"insecure"`
		Timeout         *string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	}

	if c == nil {
		*c = Connection{}
	}

	return alias{
		Host:            c.Host,
		Port:            c.Port,
		ReadBufferSize:  tc.BytesSize(float64(c.ReadBufferSize)),
		WriteBufferSize: tc.BytesSize(float64(c.WriteBufferSize)),
		MaxMessageSize:  tc.BytesSize(float64(c.MaxMessageSize)),
		Insecure:        c.Insecure,
		Timeout:         tc.String(ConvertDurationToStr(c.Timeout)),
	}, nil
}

func (c *Connection) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		Host            string  `yaml:"host" json:"host"`
		Port            uint16  `yaml:"port" json:"port"`
		ReadBufferSize  string  `yaml:"readBufferSize" json:"read_buffer_size"`
		WriteBufferSize string  `yaml:"writeBufferSize" json:"write_buffer_size"`
		MaxMessageSize  string  `yaml:"maxMessageSize" json:"max_message_size"`
		Insecure        bool    `yaml:"insecure" json:"insecure"`
		Timeout         *string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	}
	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if c == nil {
		*c = Connection{}
	}

	if tmp.Timeout != nil {
		c.Timeout, err = str2duration.ParseDuration(*tmp.Timeout)
		if err != nil {
			return err
		}
	}

	c.Host = tmp.Host
	c.Port = tmp.Port

	var tmpB int64
	if tmpB, err = tc.RAMInBytes(tmp.ReadBufferSize); err != nil {
		return err
	}

	c.ReadBufferSize = uint(tmpB)

	if tmpB, err = tc.RAMInBytes(tmp.WriteBufferSize); err != nil {
		return err
	}

	c.WriteBufferSize = uint(tmpB)

	if tmpB, err = tc.RAMInBytes(tmp.MaxMessageSize); err != nil {
		return err
	}

	c.MaxMessageSize = uint(tmpB)
	c.Insecure = tmp.Insecure

	return nil
}

type Keepalive struct {
	Time              time.Duration      `yaml:"time" json:"time" validate:"required,gt=0"`
	Timeout           time.Duration      `yaml:"timeout" json:"timeout" validate:"required,gt=0"`
	EnforcementPolicy *EnforcementPolicy `yaml:"enforcementPolicy" json:"enforcement_policy"`
}

func (ka *Keepalive) MarshalJSON() ([]byte, error) {
	type alias struct {
		Time              string             `yaml:"time" json:"time"`
		Timeout           string             `yaml:"timeout" json:"timeout"`
		EnforcementPolicy *EnforcementPolicy `yaml:"enforcementPolicy" json:"enforcement_policy"`
	}

	if ka == nil {
		*ka = Keepalive{}
	}

	return json.Marshal(alias{
		Time:              ConvertDurationToStr(ka.Time),
		Timeout:           ConvertDurationToStr(ka.Timeout),
		EnforcementPolicy: ka.EnforcementPolicy,
	})
}

func (ka *Keepalive) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		Time              string             `yaml:"time" json:"time"`
		Timeout           string             `yaml:"timeout" json:"timeout"`
		EnforcementPolicy *EnforcementPolicy `yaml:"enforcementPolicy" json:"enforcement_policy"`
	}
	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if ka == nil {
		*ka = Keepalive{}
	}

	ka.Time, err = str2duration.ParseDuration(tmp.Time)
	if err != nil {
		return err
	}

	ka.Timeout, err = str2duration.ParseDuration(tmp.Timeout)
	if err != nil {
		return err
	}

	ka.EnforcementPolicy = tmp.EnforcementPolicy

	return nil
}

func (ka *Keepalive) MarshalYAML() (interface{}, error) {
	type alias struct {
		Time              string             `yaml:"time" json:"time"`
		Timeout           string             `yaml:"timeout" json:"timeout"`
		EnforcementPolicy *EnforcementPolicy `yaml:"enforcementPolicy" json:"enforcement_policy"`
	}

	if ka == nil {
		*ka = Keepalive{}
	}

	return alias{
		Time:              ConvertDurationToStr(ka.Time),
		Timeout:           ConvertDurationToStr(ka.Timeout),
		EnforcementPolicy: ka.EnforcementPolicy,
	}, nil
}

func (ka *Keepalive) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		Time              string             `yaml:"time" json:"time"`
		Timeout           string             `yaml:"timeout" json:"timeout"`
		EnforcementPolicy *EnforcementPolicy `yaml:"enforcementPolicy" json:"enforcement_policy"`
	}
	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if ka == nil {
		*ka = Keepalive{}
	}

	ka.Time, err = str2duration.ParseDuration(tmp.Time)
	if err != nil {
		return err
	}

	ka.Timeout, err = str2duration.ParseDuration(tmp.Timeout)
	if err != nil {
		return err
	}

	ka.EnforcementPolicy = tmp.EnforcementPolicy

	return nil
}

type EnforcementPolicy struct {
	MinTime             time.Duration `yaml:"minTime" json:"min_time" validate:"required,gt=0"`
	PermitWithoutStream bool          `yaml:"permitWithoutStream" json:"permit_without_stream"`
}

func (ep *EnforcementPolicy) MarshalJSON() ([]byte, error) {
	type alias struct {
		MinTime             string `yaml:"minTime" json:"min_time"`
		PermitWithoutStream bool   `yaml:"permitWithoutStream" json:"permit_without_stream"`
	}

	if ep == nil {
		*ep = EnforcementPolicy{}
	}

	return json.Marshal(alias{
		MinTime:             ConvertDurationToStr(ep.MinTime),
		PermitWithoutStream: ep.PermitWithoutStream,
	})
}

func (ep *EnforcementPolicy) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		MinTime             string `yaml:"minTime" json:"min_time"`
		PermitWithoutStream bool   `yaml:"permitWithoutStream" json:"permit_without_stream"`
	}
	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if ep == nil {
		*ep = EnforcementPolicy{}
	}

	ep.PermitWithoutStream = tmp.PermitWithoutStream
	ep.MinTime, err = str2duration.ParseDuration(tmp.MinTime)

	return err
}

func (ep *EnforcementPolicy) MarshalYAML() (interface{}, error) {
	type alias struct {
		MinTime             string `yaml:"minTime" json:"min_time"`
		PermitWithoutStream bool   `yaml:"permitWithoutStream" json:"permit_without_stream"`
	}

	if ep == nil {
		*ep = EnforcementPolicy{}
	}

	return alias{
		MinTime:             ConvertDurationToStr(ep.MinTime),
		PermitWithoutStream: ep.PermitWithoutStream,
	}, nil
}

func (ep *EnforcementPolicy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		MinTime             string `yaml:"minTime" json:"min_time"`
		PermitWithoutStream bool   `yaml:"permitWithoutStream" json:"permit_without_stream"`
	}
	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if ep == nil {
		*ep = EnforcementPolicy{}
	}

	ep.PermitWithoutStream = tmp.PermitWithoutStream
	ep.MinTime, err = str2duration.ParseDuration(tmp.MinTime)

	return err
}

type NetHTTPTransport struct {
	KeepAlive              time.Duration `yaml:"keepAlive,omitempty" json:"keep_alive,omitempty"`
	TLSHandshakeTimeout    time.Duration `yaml:"tlsHandshakeTimeout,omitempty" json:"tls_handshake_timeout,omitempty"`
	DialTimeout            time.Duration `yaml:"dialTimeout,omitempty" json:"dial_timeout,omitempty"`
	ResponseHeaderTimeout  time.Duration `yaml:"responseHeaderTimeout,omitempty" json:"response_header_timeout,omitempty"`
	ExpectContinueTimeout  time.Duration `yaml:"expectContinueTimeout,omitempty" json:"expect_continue_timeout,omitempty"`
	DisableKeepAlives      bool          `yaml:"disableKeepAlives" json:"disable_Keep_alives"`
	DisableCompression     bool          `yaml:"disableCompression" json:"disable_compression"`
	MaxIdleConns           int           `yaml:"maxIdleConns,omitempty" json:"max_idle_conns,omitempty"`
	MaxIdleConnsPerHost    int           `yaml:"maxIdleConnsPerHost,omitempty" json:"max_idle_conns_per_host,omitempty"`
	MaxConnsPerHost        int           `yaml:"maxConnsPerHost,omitempty" json:"max_conns_per_host,omitempty"`
	MaxResponseHeaderBytes int64         `yaml:"maxResponseHeaderBytes,omitempty" json:"max_response_header_bytes,omitempty"`
	Buffer                 *Buffer       `yaml:"buffer,omitempty" json:"buffer,omitempty"`
}

func (t *NetHTTPTransport) MarshalJSON() ([]byte, error) {
	type alias struct {
		KeepAlive              string  `yaml:"keepAlive,omitempty" json:"keep_alive,omitempty"`
		TLSHandshakeTimeout    string  `yaml:"tlsHandshakeTimeout,omitempty" json:"tls_handshake_timeout,omitempty"`
		DialTimeout            string  `yaml:"dialTimeout,omitempty" json:"dial_timeout,omitempty"`
		ResponseHeaderTimeout  string  `yaml:"responseHeaderTimeout,omitempty" json:"response_header_timeout,omitempty"`
		ExpectContinueTimeout  string  `yaml:"expectContinueTimeout,omitempty" json:"expect_continue_timeout,omitempty"`
		DisableKeepAlives      bool    `yaml:"disableKeepAlives" json:"disable_Keep_alives"`
		DisableCompression     bool    `yaml:"disableCompression" json:"disable_compression"`
		MaxIdleConns           int     `yaml:"maxIdleConns,omitempty" json:"max_idle_conns,omitempty"`
		MaxIdleConnsPerHost    int     `yaml:"maxIdleConnsPerHost,omitempty" json:"max_idle_conns_per_host,omitempty"`
		MaxConnsPerHost        int     `yaml:"maxConnsPerHost,omitempty" json:"max_conns_per_host,omitempty"`
		MaxResponseHeaderBytes string  `yaml:"maxResponseHeaderBytes,omitempty" json:"max_response_header_bytes,omitempty"`
		Buffer                 *Buffer `yaml:"buffer,omitempty" json:"buffer,omitempty"`
	}

	if t == nil {
		*t = NetHTTPTransport{}
	}

	return json.Marshal(alias{
		KeepAlive:              HumanDuration(t.KeepAlive),
		TLSHandshakeTimeout:    HumanDuration(t.TLSHandshakeTimeout),
		DialTimeout:            HumanDuration(t.DialTimeout),
		ResponseHeaderTimeout:  HumanDuration(t.ResponseHeaderTimeout),
		ExpectContinueTimeout:  HumanDuration(t.ExpectContinueTimeout),
		DisableKeepAlives:      t.DisableKeepAlives,
		DisableCompression:     t.DisableCompression,
		MaxIdleConns:           t.MaxIdleConns,
		MaxIdleConnsPerHost:    t.MaxIdleConnsPerHost,
		MaxConnsPerHost:        t.MaxConnsPerHost,
		MaxResponseHeaderBytes: tc.BytesSize(float64(t.MaxResponseHeaderBytes)),
		Buffer:                 t.Buffer,
	})
}

func (t *NetHTTPTransport) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		KeepAlive              string  `yaml:"keepAlive,omitempty" json:"keep_alive,omitempty"`
		TLSHandshakeTimeout    string  `yaml:"tlsHandshakeTimeout,omitempty" json:"tls_handshake_timeout,omitempty"`
		DialTimeout            string  `yaml:"dialTimeout,omitempty" json:"dial_timeout,omitempty"`
		ResponseHeaderTimeout  string  `yaml:"responseHeaderTimeout,omitempty" json:"response_header_timeout,omitempty"`
		ExpectContinueTimeout  string  `yaml:"expectContinueTimeout,omitempty" json:"expect_continue_timeout,omitempty"`
		DisableKeepAlives      bool    `yaml:"disableKeepAlives" json:"disable_Keep_alives"`
		DisableCompression     bool    `yaml:"disableCompression" json:"disable_compression"`
		MaxIdleConns           int     `yaml:"maxIdleConns,omitempty" json:"max_idle_conns,omitempty"`
		MaxIdleConnsPerHost    int     `yaml:"maxIdleConnsPerHost,omitempty" json:"max_idle_conns_per_host,omitempty"`
		MaxConnsPerHost        int     `yaml:"maxConnsPerHost,omitempty" json:"max_conns_per_host,omitempty"`
		MaxResponseHeaderBytes string  `yaml:"maxResponseHeaderBytes,omitempty" json:"max_response_header_bytes,omitempty"`
		Buffer                 *Buffer `yaml:"buffer,omitempty" json:"buffer,omitempty"`
	}
	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if t == nil {
		*t = NetHTTPTransport{}
	}

	if len(tmp.KeepAlive) > 0 {
		t.KeepAlive, err = str2duration.ParseDuration(tmp.KeepAlive)
		if err != nil {
			return err
		}
	}

	if len(tmp.TLSHandshakeTimeout) > 0 {
		t.TLSHandshakeTimeout, err = str2duration.ParseDuration(tmp.TLSHandshakeTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.DialTimeout) > 0 {
		t.DialTimeout, err = str2duration.ParseDuration(tmp.DialTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.ResponseHeaderTimeout) > 0 {
		t.ResponseHeaderTimeout, err = str2duration.ParseDuration(tmp.ResponseHeaderTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.ExpectContinueTimeout) > 0 {
		t.ExpectContinueTimeout, err = str2duration.ParseDuration(tmp.ExpectContinueTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.MaxResponseHeaderBytes) > 0 {
		if t.MaxResponseHeaderBytes, err = tc.RAMInBytes(tmp.MaxResponseHeaderBytes); err != nil {
			return err
		}
	}

	t.DisableKeepAlives = tmp.DisableKeepAlives
	t.DisableCompression = tmp.DisableCompression
	t.MaxIdleConns = tmp.MaxIdleConns
	t.MaxIdleConnsPerHost = tmp.MaxIdleConnsPerHost
	t.MaxConnsPerHost = tmp.MaxConnsPerHost

	return nil
}

func (t *NetHTTPTransport) MarshalYAML() (interface{}, error) {
	type alias struct {
		KeepAlive              string  `yaml:"keepAlive,omitempty" json:"keep_alive,omitempty"`
		TLSHandshakeTimeout    string  `yaml:"tlsHandshakeTimeout,omitempty" json:"tls_handshake_timeout,omitempty"`
		DialTimeout            string  `yaml:"dialTimeout,omitempty" json:"dial_timeout,omitempty"`
		ResponseHeaderTimeout  string  `yaml:"responseHeaderTimeout,omitempty" json:"response_header_timeout,omitempty"`
		ExpectContinueTimeout  string  `yaml:"expectContinueTimeout,omitempty" json:"expect_continue_timeout,omitempty"`
		DisableKeepAlives      bool    `yaml:"disableKeepAlives" json:"disable_Keep_alives"`
		DisableCompression     bool    `yaml:"disableCompression" json:"disable_compression"`
		MaxIdleConns           int     `yaml:"maxIdleConns,omitempty" json:"max_idle_conns,omitempty"`
		MaxIdleConnsPerHost    int     `yaml:"maxIdleConnsPerHost,omitempty" json:"max_idle_conns_per_host,omitempty"`
		MaxConnsPerHost        int     `yaml:"maxConnsPerHost,omitempty" json:"max_conns_per_host,omitempty"`
		MaxResponseHeaderBytes string  `yaml:"maxResponseHeaderBytes,omitempty" json:"max_response_header_bytes,omitempty"`
		Buffer                 *Buffer `yaml:"buffer,omitempty" json:"buffer,omitempty"`
	}

	if t == nil {
		*t = NetHTTPTransport{}
	}

	return alias{
		KeepAlive:              HumanDuration(t.KeepAlive),
		TLSHandshakeTimeout:    HumanDuration(t.TLSHandshakeTimeout),
		DialTimeout:            HumanDuration(t.DialTimeout),
		ResponseHeaderTimeout:  HumanDuration(t.ResponseHeaderTimeout),
		ExpectContinueTimeout:  HumanDuration(t.ExpectContinueTimeout),
		DisableKeepAlives:      t.DisableKeepAlives,
		DisableCompression:     t.DisableCompression,
		MaxIdleConns:           t.MaxIdleConns,
		MaxIdleConnsPerHost:    t.MaxIdleConnsPerHost,
		MaxConnsPerHost:        t.MaxConnsPerHost,
		MaxResponseHeaderBytes: tc.BytesSize(float64(t.MaxResponseHeaderBytes)),
		Buffer:                 t.Buffer,
	}, nil
}

func (t *NetHTTPTransport) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		KeepAlive              string  `yaml:"keepAlive,omitempty" json:"keep_alive,omitempty"`
		TLSHandshakeTimeout    string  `yaml:"tlsHandshakeTimeout,omitempty" json:"tls_handshake_timeout,omitempty"`
		DialTimeout            string  `yaml:"dialTimeout,omitempty" json:"dial_timeout,omitempty"`
		ResponseHeaderTimeout  string  `yaml:"responseHeaderTimeout,omitempty" json:"response_header_timeout,omitempty"`
		ExpectContinueTimeout  string  `yaml:"expectContinueTimeout,omitempty" json:"expect_continue_timeout,omitempty"`
		DisableKeepAlives      bool    `yaml:"disableKeepAlives" json:"disable_Keep_alives"`
		DisableCompression     bool    `yaml:"disableCompression" json:"disable_compression"`
		MaxIdleConns           int     `yaml:"maxIdleConns,omitempty" json:"max_idle_conns,omitempty"`
		MaxIdleConnsPerHost    int     `yaml:"maxIdleConnsPerHost,omitempty" json:"max_idle_conns_per_host,omitempty"`
		MaxConnsPerHost        int     `yaml:"maxConnsPerHost,omitempty" json:"max_conns_per_host,omitempty"`
		MaxResponseHeaderBytes string  `yaml:"maxResponseHeaderBytes,omitempty" json:"max_response_header_bytes,omitempty"`
		Buffer                 *Buffer `yaml:"buffer,omitempty" json:"buffer,omitempty"`
	}
	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if t == nil {
		*t = NetHTTPTransport{}
	}

	if len(tmp.KeepAlive) > 0 {
		t.KeepAlive, err = str2duration.ParseDuration(tmp.KeepAlive)
		if err != nil {
			return err
		}
	}

	if len(tmp.TLSHandshakeTimeout) > 0 {
		t.TLSHandshakeTimeout, err = str2duration.ParseDuration(tmp.TLSHandshakeTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.DialTimeout) > 0 {
		t.DialTimeout, err = str2duration.ParseDuration(tmp.DialTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.ResponseHeaderTimeout) > 0 {
		t.ResponseHeaderTimeout, err = str2duration.ParseDuration(tmp.ResponseHeaderTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.ExpectContinueTimeout) > 0 {
		t.ExpectContinueTimeout, err = str2duration.ParseDuration(tmp.ExpectContinueTimeout)
		if err != nil {
			return err
		}
	}

	if len(tmp.MaxResponseHeaderBytes) > 0 {
		if t.MaxResponseHeaderBytes, err = tc.RAMInBytes(tmp.MaxResponseHeaderBytes); err != nil {
			return err
		}
	}

	t.DisableKeepAlives = tmp.DisableKeepAlives
	t.DisableCompression = tmp.DisableCompression
	t.MaxIdleConns = tmp.MaxIdleConns
	t.MaxIdleConnsPerHost = tmp.MaxIdleConnsPerHost
	t.MaxConnsPerHost = tmp.MaxConnsPerHost

	return nil
}

type HTTPTransport struct {
	MaxIdleConnDuration time.Duration     `yaml:"maxIdleConnDuration" json:"max_idle_conn_duration"`
	ReadTimeout         time.Duration     `yaml:"readTimeout" json:"read_timeout" validate:"required,gt=0"`
	WriteTimeout        time.Duration     `yaml:"writeTimeout" json:"write_timeout" validate:"required,gt=0"`
	NetTransport        *NetHTTPTransport `yaml:"netTransport,omitempty" json:"net_transport,omitempty" validate:"omitempty"`
}

func (t *HTTPTransport) GetTransportConfigs() *HTTPTransport {
	return t
}

func (t *HTTPTransport) MarshalJSON() ([]byte, error) {
	type alias struct {
		MaxIdleConnDuration string            `yaml:"maxIdleConnDuration" json:"max_idle_conn_duration"`
		ReadTimeout         string            `yaml:"readTimeout" json:"read_timeout"`
		WriteTimeout        string            `yaml:"writeTimeout" json:"write_timeout"`
		NetTransport        *NetHTTPTransport `yaml:"netTransport,omitempty" json:"net_transport,omitempty"`
	}

	if t == nil {
		*t = HTTPTransport{}
	}

	return json.Marshal(alias{
		MaxIdleConnDuration: HumanDuration(t.MaxIdleConnDuration),
		ReadTimeout:         HumanDuration(t.ReadTimeout),
		WriteTimeout:        HumanDuration(t.WriteTimeout),
		NetTransport:        t.NetTransport,
	})
}

func (t *HTTPTransport) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		MaxIdleConnDuration string            `yaml:"maxIdleConnDuration" json:"max_idle_conn_duration"`
		ReadTimeout         string            `yaml:"readTimeout" json:"read_timeout"`
		WriteTimeout        string            `yaml:"writeTimeout" json:"write_timeout"`
		NetTransport        *NetHTTPTransport `yaml:"netTransport,omitempty" json:"net_transport,omitempty"`
	}
	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if t == nil {
		*t = HTTPTransport{}
	}

	t.MaxIdleConnDuration, err = str2duration.ParseDuration(tmp.MaxIdleConnDuration)
	if err != nil {
		return err
	}

	t.ReadTimeout, err = str2duration.ParseDuration(tmp.ReadTimeout)
	if err != nil {
		return err
	}

	t.WriteTimeout, err = str2duration.ParseDuration(tmp.WriteTimeout)
	if err != nil {
		return err
	}

	t.NetTransport = tmp.NetTransport

	return nil
}

func (t *HTTPTransport) MarshalYAML() (interface{}, error) {
	type alias struct {
		MaxIdleConnDuration string            `yaml:"maxIdleConnDuration" json:"max_idle_conn_duration"`
		ReadTimeout         string            `yaml:"readTimeout" json:"read_timeout"`
		WriteTimeout        string            `yaml:"writeTimeout" json:"write_timeout"`
		NetTransport        *NetHTTPTransport `yaml:"netTransport,omitempty" json:"net_transport,omitempty"`
	}

	if t == nil {
		*t = HTTPTransport{}
	}

	return alias{
		MaxIdleConnDuration: HumanDuration(t.MaxIdleConnDuration),
		ReadTimeout:         HumanDuration(t.ReadTimeout),
		WriteTimeout:        HumanDuration(t.WriteTimeout),
		NetTransport:        t.NetTransport,
	}, nil
}

func (t *HTTPTransport) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		MaxIdleConnDuration string            `yaml:"maxIdleConnDuration" json:"max_idle_conn_duration"`
		ReadTimeout         string            `yaml:"readTimeout" json:"read_timeout"`
		WriteTimeout        string            `yaml:"writeTimeout" json:"write_timeout"`
		NetTransport        *NetHTTPTransport `yaml:"netTransport,omitempty" json:"net_transport,omitempty"`
	}
	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if t == nil {
		*t = HTTPTransport{}
	}

	t.MaxIdleConnDuration, err = str2duration.ParseDuration(tmp.MaxIdleConnDuration)
	if err != nil {
		return err
	}

	t.ReadTimeout, err = str2duration.ParseDuration(tmp.ReadTimeout)
	if err != nil {
		return err
	}

	t.WriteTimeout, err = str2duration.ParseDuration(tmp.WriteTimeout)
	if err != nil {
		return err
	}

	t.NetTransport = tmp.NetTransport

	return nil
}

type Postgres struct {
	Username string        `validate:"ascii"`
	Password string        `validate:"ascii"`
	Database string        `validate:"required,ascii"`
	Host     string        `validate:"grpc_host"`
	Port     uint16        `validate:"required,gt=0"`
	Schema   string        `validate:"alphanum"`
	SSLMode  enums.SSLMode `validate:"required"`
	Pool     *Pool         `yaml:"pool" json:"pool" validate:"required"`
}

func (pg *Postgres) Dsn() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", pg.Host, pg.Port, pg.Username, pg.Password, pg.Database)
}

type Pool struct {
	MaxIdleConns    int           `yaml:"maxIdleConns" json:"max_idle_conns" validate:"required,gt=0"`
	MaxOpenConns    int           `yaml:"maxOpenConns" json:"max_open_conns" validate:"required,gt=0"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"conn_max_lifetime" validate:"required,gt=0"`
}

func (p *Pool) MarshalYAML() (interface{}, error) {
	type alias struct {
		MaxIdleConns    int    `yaml:"maxIdleConns" json:"max_idle_conns"`
		MaxOpenConns    int    `yaml:"maxOpenConns" json:"max_open_conns"`
		ConnMaxLifetime string `yaml:"connMaxLifetime" json:"conn_max_lifetime"`
	}

	if p == nil {
		*p = Pool{}
	}

	return alias{
		MaxIdleConns:    p.MaxIdleConns,
		MaxOpenConns:    p.MaxOpenConns,
		ConnMaxLifetime: HumanDuration(p.ConnMaxLifetime),
	}, nil
}

func (p *Pool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		MaxIdleConns    int    `yaml:"maxIdleConns" json:"max_idle_conns"`
		MaxOpenConns    int    `yaml:"maxOpenConns" json:"max_open_conns"`
		ConnMaxLifetime string `yaml:"connMaxLifetime" json:"conn_max_lifetime"`
	}
	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if p == nil {
		*p = Pool{}
	}

	p.MaxIdleConns = tmp.MaxIdleConns
	p.MaxOpenConns = tmp.MaxOpenConns

	p.ConnMaxLifetime, err = str2duration.ParseDuration(tmp.ConnMaxLifetime)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pool) MarshalJSON() ([]byte, error) {
	type alias struct {
		MaxIdleConns    int    `yaml:"maxIdleConns" json:"max_idle_conns"`
		MaxOpenConns    int    `yaml:"maxOpenConns" json:"max_open_conns"`
		ConnMaxLifetime string `yaml:"connMaxLifetime" json:"conn_max_lifetime"`
	}

	if p == nil {
		*p = Pool{}
	}

	return json.Marshal(alias{
		MaxIdleConns:    p.MaxIdleConns,
		MaxOpenConns:    p.MaxOpenConns,
		ConnMaxLifetime: HumanDuration(p.ConnMaxLifetime),
	})
}

func (p *Pool) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		MaxIdleConns    int    `yaml:"maxIdleConns" json:"max_idle_conns"`
		MaxOpenConns    int    `yaml:"maxOpenConns" json:"max_open_conns"`
		ConnMaxLifetime string `yaml:"connMaxLifetime" json:"conn_max_lifetime"`
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if p == nil {
		*p = Pool{}
	}

	p.MaxIdleConns = tmp.MaxIdleConns
	p.MaxOpenConns = tmp.MaxOpenConns

	p.ConnMaxLifetime, err = str2duration.ParseDuration(tmp.ConnMaxLifetime)
	if err != nil {
		return err
	}

	return nil
}

type HTTPClient struct {
	URL       string `validate:"url"`
	Transport *ClientTransport
}

type ClientTransport struct {
	MaxIdleConns int `yaml:"maxIdleConns" json:"max_idle_conns"`
	Buffer       *Buffer
	TCPKeepalive *TCPKeepalive `yaml:"tcpKeepalive,omitempty" json:"tcpKeepalive,omitempty" validate:"required_if=Enabled true"`
	Timeout      time.Duration
}

func (ct *ClientTransport) MarshalJSON() ([]byte, error) {
	type alias struct {
		MaxIdleConns int `yaml:"maxIdleConns" json:"max_idle_conns"`
		Buffer       *Buffer
		TCPKeepalive *TCPKeepalive `yaml:"tcpKeepalive,omitempty" json:"tcpKeepalive,omitempty"`
		Timeout      string
	}

	if ct == nil {
		*ct = ClientTransport{}
	}

	return json.Marshal(alias{
		MaxIdleConns: ct.MaxIdleConns,
		Buffer:       ct.Buffer,
		TCPKeepalive: ct.TCPKeepalive,
		Timeout:      HumanDuration(ct.Timeout),
	})
}

func (ct *ClientTransport) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		MaxIdleConns int `yaml:"maxIdleConns" json:"max_idle_conns"`
		Buffer       *Buffer
		TCPKeepalive *TCPKeepalive `yaml:"tcpKeepalive,omitempty" json:"tcpKeepalive,omitempty"`
		Timeout      string
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if ct == nil {
		*ct = ClientTransport{}
	}

	ct.MaxIdleConns = tmp.MaxIdleConns
	ct.Buffer = tmp.Buffer
	ct.TCPKeepalive = tmp.TCPKeepalive

	ct.Timeout, err = str2duration.ParseDuration(tmp.Timeout)
	if err != nil {
		return err
	}

	return nil
}

func (ct *ClientTransport) MarshalYAML() (interface{}, error) {
	type alias struct {
		MaxIdleConns int `yaml:"maxIdleConns" json:"max_idle_conns"`
		Buffer       *Buffer
		TCPKeepalive *TCPKeepalive `yaml:"tcpKeepalive,omitempty" json:"tcpKeepalive,omitempty"`
		Timeout      string
	}

	if ct == nil {
		*ct = ClientTransport{}
	}

	return alias{
		MaxIdleConns: ct.MaxIdleConns,
		Buffer:       ct.Buffer,
		TCPKeepalive: ct.TCPKeepalive,
		Timeout:      HumanDuration(ct.Timeout),
	}, nil
}

func (ct *ClientTransport) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		MaxIdleConns int `yaml:"maxIdleConns" json:"max_idle_conns"`
		Buffer       *Buffer
		TCPKeepalive *TCPKeepalive `yaml:"tcpKeepalive,omitempty" json:"tcpKeepalive,omitempty"`
		Timeout      string
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if ct == nil {
		*ct = ClientTransport{}
	}

	ct.MaxIdleConns = tmp.MaxIdleConns
	ct.Buffer = tmp.Buffer
	ct.TCPKeepalive = tmp.TCPKeepalive

	ct.Timeout, err = str2duration.ParseDuration(tmp.Timeout)
	if err != nil {
		return err
	}

	return nil
}

type Server struct {
	Host          string        `validate:"host_if_enabled"`
	Port          uint16        `validate:"port_if_enabled"`
	Keepalive     bool          `yaml:"keepalive,omitempty" json:"keepalive,omitempty"`
	RequestPause  time.Duration `yaml:"requestsPause" json:"requests_pause"`
	HTTPTransport HTTPTransport
}

func (s *Server) MarshalYAML() (interface{}, error) {
	type alias struct {
		Host          string `validate:"host_if_enabled"`
		Port          uint16 `validate:"port_if_enabled"`
		Keepalive     bool   `yaml:"keepalive,omitempty" json:"keepalive,omitempty"`
		RequestPause  string `yaml:"requestsPause" json:"requests_pause"`
		HTTPTransport HTTPTransport
	}

	if s == nil {
		*s = Server{}
	}

	return alias{
		Host:          s.Host,
		Port:          s.Port,
		Keepalive:     s.Keepalive,
		RequestPause:  HumanDuration(s.RequestPause),
		HTTPTransport: s.HTTPTransport,
	}, nil
}

func (s *Server) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		Host          string `validate:"host_if_enabled"`
		Port          uint16 `validate:"port_if_enabled"`
		Keepalive     bool   `yaml:"keepalive,omitempty" json:"keepalive,omitempty"`
		RequestPause  string `yaml:"requestsPause" json:"requests_pause"`
		HTTPTransport HTTPTransport
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if s == nil {
		*s = Server{}
	}

	s.Host = tmp.Host
	s.Port = tmp.Port
	s.Keepalive = tmp.Keepalive
	s.HTTPTransport = tmp.HTTPTransport

	s.RequestPause, err = str2duration.ParseDuration(tmp.RequestPause)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) MarshalJSON() ([]byte, error) {
	type alias struct {
		Host          string `validate:"host_if_enabled"`
		Port          uint16 `validate:"port_if_enabled"`
		Keepalive     bool   `yaml:"keepalive,omitempty" json:"keepalive,omitempty"`
		RequestPause  string `yaml:"requestsPause" json:"requests_pause"`
		HTTPTransport HTTPTransport
	}

	if s == nil {
		*s = Server{}
	}

	return json.Marshal(alias{
		Host:          s.Host,
		Port:          s.Port,
		Keepalive:     s.Keepalive,
		RequestPause:  HumanDuration(s.RequestPause),
		HTTPTransport: s.HTTPTransport,
	})
}

func (s *Server) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		Host          string `validate:"host_if_enabled"`
		Port          uint16 `validate:"port_if_enabled"`
		Keepalive     bool   `yaml:"keepalive,omitempty" json:"keepalive,omitempty"`
		RequestPause  string `yaml:"requestsPause" json:"requests_pause"`
		HTTPTransport HTTPTransport
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if s == nil {
		*s = Server{}
	}

	s.Host = tmp.Host
	s.Port = tmp.Port
	s.Keepalive = tmp.Keepalive
	s.HTTPTransport = tmp.HTTPTransport

	s.RequestPause, err = str2duration.ParseDuration(tmp.RequestPause)
	if err != nil {
		return err
	}

	return nil
}
