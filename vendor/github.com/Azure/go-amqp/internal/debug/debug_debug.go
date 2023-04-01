//go:build debug
// +build debug

package debug

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

var (
	debugLevel = 1
	logger     = log.New(os.Stderr, "", log.Lmicroseconds)
)

func init() {
	level, err := strconv.Atoi(os.Getenv("DEBUG_LEVEL"))
	if err != nil {
		return
	}

	debugLevel = level
}

// Log writes the formatted string to stderr.
// Level indicates the verbosity of the messages to log.
// The greater the value, the more verbose messages will be logged.
func Log(level int, format string, v ...any) {
	if level <= debugLevel {
		logger.Printf(format, v...)
	}
}

// Assert panics if the specified condition is false.
func Assert(condition bool) {
	if !condition {
		panic("assertion failed!")
	}
}

// Assert panics with the provided message if the specified condition is false.
func Assertf(condition bool, msg string, v ...any) {
	if !condition {
		panic(fmt.Sprintf(msg, v...))
	}
}
