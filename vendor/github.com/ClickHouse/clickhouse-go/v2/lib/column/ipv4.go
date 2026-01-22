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
	"encoding/binary"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"net"
	"net/netip"
	"reflect"
)

type IPv4 struct {
	name string
	col  proto.ColIPv4
}

func (col *IPv4) Reset() {
	col.col.Reset()
}

func (col *IPv4) Name() string {
	return col.name
}

func (col *IPv4) Type() Type {
	return "IPv4"
}

func (col *IPv4) ScanType() reflect.Type {
	return scanTypeIP
}

func (col *IPv4) Rows() int {
	return col.col.Rows()
}

func (col *IPv4) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *IPv4) ScanRow(dest any, row int) error {
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
	case *uint32:
		ipV4 := col.row(row).To4()
		if ipV4 == nil {
			return &ColumnConverterError{
				Op:   "ScanRow",
				To:   fmt.Sprintf("%T", dest),
				From: "IPv4",
			}
		}
		*d = binary.BigEndian.Uint32(ipV4[:])
	case **uint32:
		ipV4 := col.row(row).To4()
		if ipV4 == nil {
			return &ColumnConverterError{
				Op:   "ScanRow",
				To:   fmt.Sprintf("%T", dest),
				From: "IPv4",
			}
		}
		*d = new(uint32)
		**d = binary.BigEndian.Uint32(ipV4[:])
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "IPv4",
		}
	}
	return nil
}

func strToIPV4(strIp string) (netip.Addr, error) {
	ip, err := netip.ParseAddr(strIp)
	if err != nil {
		return netip.Addr{}, &ColumnConverterError{
			Op:   "Append",
			To:   "IPv4",
			Hint: "invalid IP format",
		}
	}
	return ip, nil
}

func (col *IPv4) AppendV4IPs(ips []netip.Addr) {
	for i := range ips {
		col.col.Append(proto.ToIPv4(ips[i]))
	}
}

func (col *IPv4) Append(v any) (nulls []uint8, err error) {

	switch v := v.(type) {
	case []string:
		nulls = make([]uint8, len(v))
		ips := make([]netip.Addr, len(v), len(v))
		for i := range v {
			ip, err := strToIPV4(v[i])
			if err != nil {
				return nulls, err
			}
			ips[i] = ip
		}
		col.AppendV4IPs(ips)
	case []*string:
		nulls = make([]uint8, len(v))
		ips := make([]netip.Addr, len(v), len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				ip, err := strToIPV4(*v[i])
				if err != nil {
					return nulls, err
				}
				ips[i] = ip
			default:
				ips[i] = netip.Addr{}
				nulls[i] = 1
			}
		}
		col.AppendV4IPs(ips)
	case []netip.Addr:
		nulls = make([]uint8, len(v))
		col.AppendV4IPs(v)
	case []*netip.Addr:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(proto.ToIPv4(*v[i]))
			default:
				nulls[i] = 1
				col.col.Append(0)
			}
		}
	case []net.IP:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(proto.ToIPv4(netIPToNetIPAddr(v[i])))
		}
	case []*net.IP:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(proto.ToIPv4(netIPToNetIPAddr(*v[i])))
			default:
				nulls[i] = 1
				col.col.Append(0)
			}
		}
	case []uint32:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(proto.IPv4(v[i]))
		}
	case []*uint32:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(proto.IPv4(*v[i]))
			default:
				nulls[i] = 1
				col.col.Append(0)
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "IPv4",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "IPv4",
			From: fmt.Sprintf("%T", v),
		}
	}

	return
}

func (col *IPv4) AppendRow(v any) (err error) {
	switch v := v.(type) {
	case string:
		ip, err := strToIPV4(v)
		if err != nil {
			return err
		}
		col.col.Append(proto.ToIPv4(ip))
	case *string:
		switch {
		case v != nil:
			ip, err := strToIPV4(*v)
			if err != nil {
				return err
			}
			col.col.Append(proto.ToIPv4(ip))
		default:
			col.col.Append(0)
		}
	case netip.Addr:
		col.col.Append(proto.ToIPv4(v))
	case *netip.Addr:
		switch {
		case v != nil:
			col.col.Append(proto.ToIPv4(*v))
		default:
			col.col.Append(0)
		}
	case net.IP:
		switch {
		case len(v) == 0:
			col.col.Append(0)
		default:
			col.col.Append(proto.ToIPv4(netIPToNetIPAddr(v)))
		}
	case *net.IP:
		switch {
		case v != nil:
			col.col.Append(proto.ToIPv4(netIPToNetIPAddr(*v)))
		default:
			col.col.Append(0)
		}
	case nil:
		col.col.Append(0)
	case uint32:
		col.col.Append(proto.IPv4(v))
	case *uint32:
		switch {
		case v != nil:
			col.col.Append(proto.IPv4(*v))
		default:
			col.col.Append(0)
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "IPv4",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "IPv4",
			From: fmt.Sprintf("%T", v),
		}
	}

	return
}

func (col *IPv4) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *IPv4) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

// TODO: This should probably return an netip.Addr
func (col *IPv4) row(i int) net.IP {
	src := col.col.Row(i).ToIP()
	ip := src.As4()
	return net.IPv4(ip[0], ip[1], ip[2], ip[3]).To4()
}

func (col *IPv4) rowAddr(i int) netip.Addr {
	return col.col.Row(i).ToIP()
}

func netIPToNetIPAddr(ip net.IP) netip.Addr {
	switch len(ip) {
	case 4:
		return netip.AddrFrom4([4]byte{ip[0], ip[1], ip[2], ip[3]})
	case 16:
		return netip.AddrFrom4([4]byte{ip[12], ip[13], ip[14], ip[15]})
	}
	return netip.Addr{}
}

var _ Interface = (*IPv4)(nil)
