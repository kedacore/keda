// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package column

import (
	"database/sql/driver"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"net"
	"net/netip"
	"reflect"
)

type IPv6 struct {
	col  proto.ColIPv6
	name string
}

func (col *IPv6) Reset() {
	col.col.Reset()
}

func (col *IPv6) Name() string {
	return col.name
}

func (col *IPv6) Type() Type {
	return "IPv6"
}

func (col *IPv6) ScanType() reflect.Type {
	return scanTypeIP
}

func (col *IPv6) Rows() int {
	return col.col.Rows()
}

func (col *IPv6) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *IPv6) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *string:
		*d = col.row(row).String()
	case **string:
		*d = new(string)
		**d = col.row(row).String()
	case *net.IP:
		*d = col.row(row)
	case **net.IP:
		*d = new(net.IP)
		**d = col.row(row)
	case *netip.Addr:
		*d = col.rowAddr(row)
	case **netip.Addr:
		*d = new(netip.Addr)
		**d = col.rowAddr(row)
	case *[]byte:
		*d = col.row(row)
	case **[]byte:
		*d = new([]byte)
		**d = col.row(row)
	case *proto.IPv6:
		*d = col.col.Row(row)
	case **proto.IPv6:
		*d = new(proto.IPv6)
		**d = col.col.Row(row)
	case *[16]byte:
		*d = col.col.Row(row)
	case **[16]byte:
		*d = new([16]byte)
		**d = col.col.Row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "IPv6",
		}
	}
	return nil
}

func strToIPV6(strIp string) (netip.Addr, error) {
	ip, err := netip.ParseAddr(strIp)
	if err != nil {
		return netip.Addr{}, err
	}
	return ip, nil
}

func (col *IPv6) AppendV6IPs(ips []netip.Addr) {
	for i := range ips {
		col.col.Append(proto.ToIPv6(ips[i]))
	}
}

func (col *IPv6) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []string:
		nulls = make([]uint8, len(v))
		ips := make([]netip.Addr, len(v), len(v))
		for i := range v {
			ip, err := strToIPV6(v[i])
			if err != nil {
				return nulls, &ColumnConverterError{
					Op:   "Append",
					To:   "IPv6",
					Hint: "invalid IP format",
				}
			}
			ips[i] = ip
		}
		col.AppendV6IPs(ips)
	case []*string:
		nulls = make([]uint8, len(v))
		ips := make([]netip.Addr, len(v), len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				ip, err := strToIPV6(*v[i])
				if err != nil {
					return nulls, &ColumnConverterError{
						Op:   "Append",
						To:   "IPv6",
						Hint: "invalid IP format",
					}
				}
				ips[i] = ip
			default:
				ips[i] = netip.Addr{}
				nulls[i] = 1
			}
		}
		col.AppendV6IPs(ips)
	case []netip.Addr:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			col.col.Append(proto.ToIPv6(v))
		}
	case []*netip.Addr:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				col.col.Append(proto.ToIPv6(*v))
			default:
				nulls[i] = 1
				col.col.Append([16]byte{})
			}
		}
	case []net.IP:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(v))))
		}
	case []*net.IP:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(*v))))
			default:
				nulls[i] = 1
				col.col.Append([16]byte{})
			}
		}
	case [][]byte:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(v))))
		}
	case []*[]byte:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(*v))))
			default:
				nulls[i] = 1
				col.col.Append([16]byte{})
			}
		}
	case [][16]byte:
		for _, v := range v {
			col.col.Append(v)
		}
	case []*[16]byte:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				col.col.Append(*v)
			default:
				nulls[i] = 1
				col.col.Append([16]byte{})
			}
		}
	case []proto.IPv6:
		for _, v := range v {
			col.col.Append(v)
		}
	case []*proto.IPv6:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				col.col.Append(*v)
			default:
				nulls[i] = 1
				col.col.Append([16]byte{})
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "IPv6",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "IPv6",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *IPv6) AppendRow(v any) (err error) {
	switch v := v.(type) {
	case string:
		ip, err := strToIPV6(v)
		if err != nil {
			return &ColumnConverterError{
				Op:   "Append",
				To:   "IPv6",
				Hint: "invalid IP format",
			}
		}
		col.col.Append(ip.As16())
	case *string:
		switch {
		case v != nil:
			ip, err := strToIPV6(*v)
			if err != nil {
				return &ColumnConverterError{
					Op:   "Append",
					To:   "IPv6",
					Hint: "invalid IP format",
				}
			}
			col.col.Append(ip.As16())
		default:
			col.col.Append([16]byte{})
		}
	case netip.Addr:
		col.col.Append(proto.ToIPv6(v))
	case *netip.Addr:
		switch {
		case v != nil:
			col.col.Append(proto.ToIPv6(*v))
		default:
			col.col.Append([16]byte{})
		}
	case net.IP:
		col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(v))))
	case *net.IP:
		switch {
		case v != nil:
			col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(*v))))
		default:
			col.col.Append([16]byte{})
		}
	case []byte:
		col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(v))))
	case *[]byte:
		switch {
		case v != nil:
			col.col.Append(proto.ToIPv6(netip.AddrFrom16(IPv6ToBytes(*v))))
		default:
			col.col.Append([16]byte{})
		}
	case [16]byte:
		col.col.Append(v)
	case *[16]byte:
		switch {
		case v != nil:
			col.col.Append(*v)
		default:
			col.col.Append([16]byte{})
		}
	case proto.IPv6:
		col.col.Append(v)
	case *proto.IPv6:
		switch {
		case v != nil:
			col.col.Append(*v)
		default:
			col.col.Append([16]byte{})
		}
	case nil:
		col.col.Append([16]byte{})
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "IPv6",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "IPv6",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *IPv6) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *IPv6) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func IPv6ToBytes(ip net.IP) [16]byte {
	if ip == nil {
		return [16]byte{}
	}

	if len(ip) == 4 {
		ip = ip.To16()
	}
	return [16]byte{ip[0], ip[1], ip[2], ip[3], ip[4], ip[5], ip[6], ip[7], ip[8], ip[9], ip[10], ip[11], ip[12], ip[13], ip[14], ip[15]}
}

// TODO: This should probably return an netip.Addr
func (col *IPv6) row(i int) net.IP {
	src := col.col.Row(i)
	return src[:]
}

func (col *IPv6) rowAddr(i int) netip.Addr {
	return col.col.Row(i).ToIP()
}

var _ Interface = (*IPv6)(nil)
