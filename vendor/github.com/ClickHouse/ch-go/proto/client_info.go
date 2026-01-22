package proto

import (
	"github.com/go-faster/errors"
	"github.com/segmentio/asm/bswap"
	"go.opentelemetry.io/otel/trace"
)

//go:generate go run github.com/dmarkham/enumer -type Interface -trimprefix Interface -output client_info_interface_enum.go

// Interface is interface of client.
type Interface byte

// Possible interfaces.
const (
	InterfaceTCP  Interface = 1
	InterfaceHTTP Interface = 2
)

//go:generate go run github.com/dmarkham/enumer -type ClientQueryKind -trimprefix ClientQueryKind -output client_info_query_enum.go

// ClientQueryKind is kind of query.
type ClientQueryKind byte

// Possible query kinds.
const (
	ClientQueryNone      ClientQueryKind = 0
	ClientQueryInitial   ClientQueryKind = 1
	ClientQuerySecondary ClientQueryKind = 2
)

// ClientInfo message.
type ClientInfo struct {
	ProtocolVersion int

	Major int
	Minor int
	Patch int

	Interface Interface
	Query     ClientQueryKind

	InitialUser    string
	InitialQueryID string
	InitialAddress string
	InitialTime    int64

	OSUser         string
	ClientHostname string
	ClientName     string

	Span trace.SpanContext

	QuotaKey         string
	DistributedDepth int

	// For parallel processing on replicas.

	CollaborateWithInitiator   bool
	CountParticipatingReplicas int
	NumberOfCurrentReplica     int
}

// EncodeAware encodes to buffer version-aware.
func (c ClientInfo) EncodeAware(b *Buffer, version int) {
	b.PutByte(byte(c.Query))

	b.PutString(c.InitialUser)
	b.PutString(c.InitialQueryID)
	b.PutString(c.InitialAddress)
	if FeatureQueryStartTime.In(version) {
		b.PutInt64(c.InitialTime)
	}

	b.PutByte(byte(c.Interface))

	b.PutString(c.OSUser)
	b.PutString(c.ClientHostname)
	b.PutString(c.ClientName)

	b.PutInt(c.Major)
	b.PutInt(c.Minor)
	b.PutInt(c.ProtocolVersion)

	if FeatureQuotaKeyInClientInfo.In(version) {
		b.PutString(c.QuotaKey)
	}
	if FeatureDistributedDepth.In(version) {
		b.PutInt(c.DistributedDepth)
	}
	if FeatureVersionPatch.In(version) && c.Interface == InterfaceTCP {
		b.PutInt(c.Patch)
	}
	if FeatureOpenTelemetry.In(version) {
		if c.Span.IsValid() {
			b.PutByte(1)
			{
				v := c.Span.TraceID()
				start := len(b.Buf)
				b.Buf = append(b.Buf, v[:]...)
				bswap.Swap64(b.Buf[start:]) // https://github.com/ClickHouse/ClickHouse/issues/34369
			}
			{
				v := c.Span.SpanID()
				start := len(b.Buf)
				b.Buf = append(b.Buf, v[:]...)
				bswap.Swap64(b.Buf[start:]) // https://github.com/ClickHouse/ClickHouse/issues/34369
			}
			b.PutString(c.Span.TraceState().String())
			b.PutByte(byte(c.Span.TraceFlags()))
		} else {
			// No OTEL data.
			b.PutByte(0)
		}
	}
	if FeatureParallelReplicas.In(version) {
		if c.CollaborateWithInitiator {
			b.PutInt(1)
		} else {
			b.PutInt(0)
		}
		b.PutInt(c.CountParticipatingReplicas)
		b.PutInt(c.NumberOfCurrentReplica)
	}
}

func (c *ClientInfo) DecodeAware(r *Reader, version int) error {
	{
		v, err := r.UInt8()
		if err != nil {
			return errors.Wrap(err, "query kind")
		}
		c.Query = ClientQueryKind(v)
		if !c.Query.IsAClientQueryKind() {
			return errors.Errorf("unknown query kind %d", v)
		}
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "initial user")
		}
		c.InitialUser = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "initial query id")
		}
		c.InitialQueryID = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "initial address")
		}
		c.InitialAddress = v
	}

	if FeatureQueryStartTime.In(version) {
		// Microseconds.
		v, err := r.Int64()
		if err != nil {
			return errors.Wrap(err, "query start time")
		}
		c.InitialTime = v
	}

	{
		v, err := r.UInt8()
		if err != nil {
			return errors.Wrap(err, "interface")
		}
		c.Interface = Interface(v)
		if !c.Interface.IsAInterface() {
			return errors.Errorf("unknown interface %d", v)
		}

		// TODO(ernado): support HTTP
		if c.Interface != InterfaceTCP {
			return errors.New("only tcp interface is supported")
		}
	}

	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "os user")
		}
		c.OSUser = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "client hostname")
		}
		c.ClientHostname = v
	}
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "client name")
		}
		c.ClientName = v
	}

	{
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "major version")
		}
		c.Major = v
	}
	{
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "minor version")
		}
		c.Minor = v
	}
	{
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "protocol version")
		}
		c.ProtocolVersion = v
	}

	if FeatureQuotaKeyInClientInfo.In(version) {
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "quota key")
		}
		c.QuotaKey = v
	}
	if FeatureDistributedDepth.In(version) {
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "distributed depth")
		}
		c.DistributedDepth = v
	}
	if FeatureVersionPatch.In(version) && c.Interface == InterfaceTCP {
		v, err := r.Int()
		if err != nil {
			return errors.Wrap(err, "patch version")
		}
		c.Patch = v
	}
	if FeatureOpenTelemetry.In(version) {
		hasTrace, err := r.Bool()
		if err != nil {
			return errors.Wrap(err, "open telemetry start")
		}
		if hasTrace {
			var cfg trace.SpanContextConfig
			{
				v, err := r.ReadRaw(len(cfg.TraceID))
				if err != nil {
					return errors.Wrap(err, "trace id")
				}
				bswap.Swap64(v) // https://github.com/ClickHouse/ClickHouse/issues/34369
				copy(cfg.TraceID[:], v)
			}
			{
				v, err := r.ReadRaw(len(cfg.SpanID))
				if err != nil {
					return errors.Wrap(err, "span id")
				}
				bswap.Swap64(v) // https://github.com/ClickHouse/ClickHouse/issues/34369
				copy(cfg.SpanID[:], v)
			}
			{
				v, err := r.Str()
				if err != nil {
					return errors.Wrap(err, "trace state")
				}
				state, err := trace.ParseTraceState(v)
				if err != nil {
					return errors.Wrap(err, "parse trace state")
				}
				cfg.TraceState = state
			}
			{
				v, err := r.Byte()
				if err != nil {
					return errors.Wrap(err, "trace flag")
				}
				cfg.TraceFlags = trace.TraceFlags(v)
			}
			c.Span = trace.NewSpanContext(cfg)
		}
	}
	if FeatureParallelReplicas.In(version) {
		{
			v, err := r.Int()
			if err != nil {
				return errors.Wrap(err, "parallel replicas")
			}
			c.CollaborateWithInitiator = v == 1
		}
		{
			v, err := r.Int()
			if err != nil {
				return errors.Wrap(err, "count participating replicas")
			}
			c.CountParticipatingReplicas = v
		}
		{
			v, err := r.Int()
			if err != nil {
				return errors.Wrap(err, "number of current replica")
			}
			c.NumberOfCurrentReplica = v
		}
	}

	return nil
}
