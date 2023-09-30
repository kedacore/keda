package customizations

import (
	"time"
)

// ReadTimeoutDuration is the read timeout that will be set for certain kinesis operations.
var ReadTimeoutDuration = 5 * time.Second
