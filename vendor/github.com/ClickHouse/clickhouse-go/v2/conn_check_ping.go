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
