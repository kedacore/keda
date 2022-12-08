package http_transport

import "time"

type requestStats struct {
	connStart time.Time
	connEnd   time.Time
	reqStart  time.Time
	reqEnd    time.Time
}
