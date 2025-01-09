package beanstalk

import "errors"

// ConnError records an error message from the server and the operation
// and connection that caused it.
type ConnError struct {
	Conn *Conn
	Op   string
	Err  error
}

func (e ConnError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

func (e ConnError) Unwrap() error {
	return e.Err
}

// Error messages returned by the server.
var (
	ErrBadFormat  = errors.New("bad command format")
	ErrBuried     = errors.New("buried")
	ErrDeadline   = errors.New("deadline soon")
	ErrDraining   = errors.New("draining")
	ErrInternal   = errors.New("internal error")
	ErrJobTooBig  = errors.New("job too big")
	ErrNoCRLF     = errors.New("expected CR LF")
	ErrNotFound   = errors.New("not found")
	ErrNotIgnored = errors.New("not ignored")
	ErrOOM        = errors.New("server is out of memory")
	ErrTimeout    = errors.New("timeout")
	ErrUnknown    = errors.New("unknown command")
)

var respError = map[string]error{
	"BAD_FORMAT":      ErrBadFormat,
	"BURIED":          ErrBuried,
	"DEADLINE_SOON":   ErrDeadline,
	"DRAINING":        ErrDraining,
	"EXPECTED_CRLF":   ErrNoCRLF,
	"INTERNAL_ERROR":  ErrInternal,
	"JOB_TOO_BIG":     ErrJobTooBig,
	"NOT_FOUND":       ErrNotFound,
	"NOT_IGNORED":     ErrNotIgnored,
	"OUT_OF_MEMORY":   ErrOOM,
	"TIMED_OUT":       ErrTimeout,
	"UNKNOWN_COMMAND": ErrUnknown,
}

type unknownRespError string

func (e unknownRespError) Error() string {
	return "unknown response: " + string(e)
}

func findRespError(s string) error {
	if err := respError[s]; err != nil {
		return err
	}
	return unknownRespError(s)
}
