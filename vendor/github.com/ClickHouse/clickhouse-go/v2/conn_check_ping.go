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

//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !solaris && !illumos
// +build !linux,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd,!solaris,!illumos

package clickhouse

import (
	"context"
	"time"
)

func (c *connect) connCheck() error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()
	if err := c.ping(ctx); err != nil {
		return err
	}
	return nil
}
