package log

import (
	"log"
	"strings"
	"sync/atomic"
)

var (
	unsafeWarningIssued int32
)

const (
	unsafeWarning = `
KUSTO UNSAFE SETIINGS HAVE BEEN EXPLICITLY ENABLED. If this has not been done on purpose, remove the import
of the kusto/unsafe package and all reference to functions or methods with "Unsafe" in their title. Unsafe
methods have the potential for SQL-like injection attacks from service clients. If you still intend to use
unsafe methods, be sure to scrub your data inputs before sending them to Kusto. This message can be suppressed.
`
)

// UnsafeWarning prints to the log a warning that unsafe methods are being used. UnsafeWarning() should be
// called on all methods or constructors that do something unsafe. The warning will only issue on the
// first call.
func UnsafeWarning(suppress bool) {
	if !suppress {
		if atomic.CompareAndSwapInt32(&unsafeWarningIssued, 0, 1) {
			log.Println(strings.TrimSpace(unsafeWarning))
		}
	}
}
