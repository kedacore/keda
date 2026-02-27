package proto

import (
	"encoding/binary"
	"net/netip"
)

// IPv4 represents IPv4 address as uint32 number.
//
// Not using netip.Addr because uint32 is 5 times faster,
// consumes 6 times less memory and better represents IPv4.
//
// Use ToIP helper for convenience.
type IPv4 uint32

func (v IPv4) String() string {
	return v.ToIP().String()
}

// ToIP represents IPv4 as netaddr.IP.
func (v IPv4) ToIP() netip.Addr {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(v))
	return netip.AddrFrom4(buf)
}

// ToIPv4 represents ip as IPv4. Panics if ip is not ipv4.
func ToIPv4(ip netip.Addr) IPv4 {
	b := ip.As4()
	return IPv4(binary.BigEndian.Uint32(b[:]))
}
