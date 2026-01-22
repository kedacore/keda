package proto

import (
	"net/netip"
)

// IPv6 represents IPv6 address.
//
// Same as FixedString(16) internally in ClickHouse.
type IPv6 [16]byte

func (v IPv6) String() string {
	return v.ToIP().String()
}

// ToIP represents IPv6 as netip.IP.
func (v IPv6) ToIP() netip.Addr {
	return netip.AddrFrom16(v)
}

// ToIPv6 represents ip as IPv6.
func ToIPv6(ip netip.Addr) IPv6 { return ip.As16() }

func binIPv6(b []byte) IPv6       { return *(*[16]byte)(b) }
func binPutIPv6(b []byte, v IPv6) { copy(b, v[:]) }
