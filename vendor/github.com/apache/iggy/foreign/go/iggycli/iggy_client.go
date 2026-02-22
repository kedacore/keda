// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package iggycli

import (
	"fmt"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
	"github.com/apache/iggy/foreign/go/tcp"
)

type Options struct {
	protocol   iggcon.Protocol
	tcpOptions []tcp.Option
}

func GetDefaultOptions() Options {
	return Options{
		protocol:   iggcon.Tcp,
		tcpOptions: nil,
	}
}

type Option func(*Options)

// WithTcp sets the client protocol to TCP and applies custom TCP options.
func WithTcp(tcpOpts ...tcp.Option) Option {
	return func(opts *Options) {
		opts.protocol = iggcon.Tcp
		opts.tcpOptions = tcpOpts
	}
}

// NewIggyClient create the IggyClient instance.
// If no Option is provided, NewIggyClient will create a default TCP client.
func NewIggyClient(options ...Option) (Client, error) {
	opts := GetDefaultOptions()

	for _, opt := range options {
		opt(&opts)
	}

	var err error
	var cli Client
	switch opts.protocol {
	case iggcon.Tcp:
		cli, err = tcp.NewIggyTcpClient(opts.tcpOptions...)
	default:
		return nil, fmt.Errorf("unknown protocol type: %v", opts.protocol)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create an iggy client: %w", err)
	}

	return cli, nil
}
