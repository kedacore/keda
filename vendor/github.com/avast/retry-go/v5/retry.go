/*
Simple library for retry mechanism

Slightly inspired by [Try::Tiny::Retry](https://metacpan.org/pod/Try::Tiny::Retry)

# SYNOPSIS

HTTP GET with retry:

	url := "http://example.com"
	var body []byte

	err := retry.New(
		retry.Attempts(5),
		retry.Delay(100*time.Millisecond),
	).Do(
		func() error {
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		// handle error
	}

	fmt.Println(string(body))

HTTP GET with retry with data:

	url := "http://example.com"

	body, err := retry.DoWithData(retry.New(),
		func() ([]byte, error) {
			resp, err := http.Get(url)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			return body, nil
		},
	)

	if err != nil {
		// handle error
	}

	fmt.Println(string(body))

Reusable retrier for high-frequency retry operations:

	// Create retrier once, reuse many times
	retrier := retry.New(
		retry.Attempts(5),
		retry.Delay(100*time.Millisecond),
	)

	// Minimal allocations in happy path
	for {
		err := retrier.Do(
			func() error {
				return doWork()
			},
		)
		if err != nil {
			// handle error
		}
	}

[More examples](https://github.com/avast/retry-go/tree/main/examples)

# SEE ALSO

* [codeGROOVE-dev/retry](https://github.com/codeGROOVE-dev/retry) - Modern fork of avast/retry-go/v4 focused on correctness, reliability and efficiency. 100% API-compatible drop-in replacement. Looks really good.

* [giantswarm/retry-go](https://github.com/giantswarm/retry-go) - slightly complicated interface.

* [sethgrid/pester](https://github.com/sethgrid/pester) - only http retry for http calls with retries and backoff

* [cenkalti/backoff](https://github.com/cenkalti/backoff) - Go port of the exponential backoff algorithm from Google's HTTP Client Library for Java. Really complicated interface.

* [rafaeljesus/retry-go](https://github.com/rafaeljesus/retry-go) - looks good, slightly similar as this package, don't have 'simple' `Retry` method

* [matryer/try](https://github.com/matryer/try) - very popular package, nonintuitive interface (for me)

# BREAKING CHANGES

* 5.0.0
  - Complete API redesign: method-based retry operations
  - Renamed `Config` type to `Retrier`
  - Renamed `NewConfig()` to `New()`
  - Changed from package-level functions to methods: `retry.Do(func, config)` → `retry.New(opts...).Do(func)`
  - `DelayTypeFunc` signature changed: `func(n uint, err error, config *Config)` → `func(n uint, err error, r *Retrier)`
  - Migration: `retry.Do(func, opts...)` → `retry.New(opts...).Do(func)` (simple find & replace)
  - This change improves performance, simplifies the API, and provides a cleaner interface
  - `Unwrap()` now returns `[]error` instead of `error` to support Go 1.20 multiple error wrapping.
  - `errors.Unwrap(err)` will now return `nil` (same as `errors.Join`). Use `errors.Is` or `errors.As` to inspect wrapped errors.

* 4.0.0
  - infinity retry is possible by set `Attempts(0)` by PR [#49](https://github.com/avast/retry-go/pull/49)

* 3.0.0
  - `DelayTypeFunc` accepts a new parameter `err` - this breaking change affects only your custom Delay Functions. This change allow [make delay functions based on error](examples/delay_based_on_error_test.go).

* 1.0.2 -> 2.0.0
  - argument of `retry.Delay` is final delay (no multiplication by `retry.Units` anymore)
  - function `retry.Units` are removed
  - [more about this breaking change](https://github.com/avast/retry-go/issues/7)

* 0.3.0 -> 1.0.0
  - `retry.Retry` function are changed to `retry.Do` function
  - `retry.RetryCustom` (OnRetry) and `retry.RetryCustomWithOpts` functions are now implement via functions produces Options (aka `retry.OnRetry`)
*/
package retry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Function signature of retryable function
type RetryableFunc func() error

// Function signature of retryable function with data
type RetryableFuncWithData[T any] func() (T, error)

// Default r.timer is a wrapper around time.After
type timerImpl struct{}

func (t *timerImpl) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Do executes the retryable function using this Retrier's configuration.
func (r *Retrier) Do(retryableFunc RetryableFunc) error {
	retryableFuncWithData := func() (any, error) {
		return nil, retryableFunc()
	}

	_, err := doWithData(r.retrierCore, retryableFuncWithData)
	return err
}

// Do executes the retryable function using this RetrierWithData's configuration.
func (r *RetrierWithData[T]) Do(retryableFunc RetryableFuncWithData[T]) (T, error) {
	return doWithData(r.retrierCore, retryableFunc)
}

func doWithData[T any](r *retrierCore, retryableFunc RetryableFuncWithData[T]) (T, error) {
	var emptyT T
	var n uint

	if err := context.Cause(r.context); err != nil {
		return emptyT, err
	}

	// Setting r.attempts to 0 means we'll retry until we succeed
	var lastErr error
	if r.attempts == 0 {
		for {
			t, err := retryableFunc()
			if err == nil {
				return t, nil
			}

			if !IsRecoverable(err) {
				return emptyT, err
			}

			if !r.retryIf(err) {
				return emptyT, err
			}

			lastErr = err

			r.onRetry(n, err)
			n++
			select {
			case <-r.timer.After(r.computeDelay(n, err)):
			case <-r.context.Done():
				if r.wrapContextErrorWithLastError {
					return emptyT, Error{context.Cause(r.context), lastErr}
				}
				return emptyT, context.Cause(r.context)
			}
		}
	}

	errorLog := Error{}

	attemptsForErrorCopy := make(map[error]uint, len(r.attemptsForError))
	for err, attempts := range r.attemptsForError {
		attemptsForErrorCopy[err] = attempts
	}

shouldRetry:
	for {
		t, err := retryableFunc()
		if err == nil {
			return t, nil
		}

		errorLog = append(errorLog, unpackUnrecoverable(err))

		if !r.retryIf(err) {
			break
		}

		r.onRetry(n, err)

		for errToCheck, attemptsForThisError := range attemptsForErrorCopy {
			if errors.Is(err, errToCheck) {
				attemptsForThisError--
				attemptsForErrorCopy[errToCheck] = attemptsForThisError
				if attemptsForThisError <= 0 {
					break shouldRetry
				}
			}
		}

		// if this is last attempt - don't wait
		if n == r.attempts-1 {
			break shouldRetry
		}
		n++
		select {
		case <-r.timer.After(r.computeDelay(n, err)):
		case <-r.context.Done():
			if r.lastErrorOnly {
				return emptyT, context.Cause(r.context)
			}

			return emptyT, append(errorLog, context.Cause(r.context))
		}
	}

	if r.lastErrorOnly {
		return emptyT, errorLog[len(errorLog)-1]
	}
	return emptyT, errorLog
}

// Error type represents list of errors in retry
type Error []error

// Error method return string representation of Error
// It is an implementation of error interface
func (e Error) Error() string {
	logWithNumber := make([]string, len(e))
	for i, l := range e {
		if l != nil {
			logWithNumber[i] = fmt.Sprintf("#%d: %s", i+1, l.Error())
		}
	}

	return fmt.Sprintf("All attempts fail:\n%s", strings.Join(logWithNumber, "\n"))
}

func (e Error) Is(target error) bool {
	for _, v := range e {
		if errors.Is(v, target) {
			return true
		}
	}
	return false
}

func (e Error) As(target interface{}) bool {
	for _, v := range e {
		if errors.As(v, target) {
			return true
		}
	}
	return false
}

// Unwrap returns the list of errors that this Error is wrapping.
//
// This method implements the Unwrap() []error interface introduced in Go 1.20
// for multi-error unwrapping. This allows errors.Is and errors.As to traverse
// all wrapped errors, not just the last one.
//
// IMPORTANT: errors.Unwrap(err) will return nil because the standard library's
// errors.Unwrap function only calls Unwrap() error, not Unwrap() []error.
// This is the same behavior as errors.Join in Go 1.20.
//
// Example - Use errors.Is to check for specific errors:
//
//	err := retry.New(retry.Attempts(3)).Do(func() error {
//		return os.ErrNotExist
//	})
//	if errors.Is(err, os.ErrNotExist) {
//		// Handle not exist error
//	}
//
// Example - Use errors.As to extract error details:
//
//	var pathErr *fs.PathError
//	if errors.As(err, &pathErr) {
//		fmt.Println("Failed at path:", pathErr.Path)
//	}
//
// Example - Get the last error directly (for migration):
//
//	if retryErr, ok := err.(retry.Error); ok {
//		lastErr := retryErr.LastError()
//	}
//
// See also: LastError() for direct access to the last error.
func (e Error) Unwrap() []error {
	return e
}

// WrappedErrors returns the list of errors that this Error is wrapping.
// It is an implementation of the `errwrap.Wrapper` interface
// in package [errwrap](https://github.com/hashicorp/errwrap) so that
// `retry.Error` can be used with that library.
func (e Error) WrappedErrors() []error {
	return e
}

// LastError returns the last error in the error list.
//
// This is a convenience method for users migrating from retry-go v4.x where
// errors.Unwrap(err) returned the last error. In v5.0.0, errors.Unwrap(err)
// returns nil due to the switch to Unwrap() []error for Go 1.20 compatibility.
//
// Migration example:
//
//	// v4.x code:
//	lastErr := errors.Unwrap(retryErr)
//
//	// v5.0.0 code (option 1 - recommended):
//	if errors.Is(retryErr, specificError) { ... }
//
//	// v5.0.0 code (option 2 - if you need the last error):
//	lastErr := retryErr.(retry.Error).LastError()
//
// Note: Using errors.Is or errors.As is preferred as they check ALL wrapped
// errors, not just the last one.
func (e Error) LastError() error {
	if len(e) == 0 {
		return nil
	}
	return e[len(e)-1]
}

type unrecoverableError struct {
	error
}

func (e unrecoverableError) Error() string {
	if e.error == nil {
		return "unrecoverable error"
	}
	return e.error.Error()
}

func (e unrecoverableError) Unwrap() error {
	return e.error
}

// Unrecoverable wraps an error in `unrecoverableError` struct
func Unrecoverable(err error) error {
	return unrecoverableError{err}
}

// IsRecoverable checks if error is an instance of `unrecoverableError`
func IsRecoverable(err error) bool {
	return !errors.Is(err, unrecoverableError{})
}

// Adds support for errors.Is usage on unrecoverableError
func (unrecoverableError) Is(err error) bool {
	_, isUnrecoverable := err.(unrecoverableError)
	return isUnrecoverable
}

func unpackUnrecoverable(err error) error {
	if unrecoverable, isUnrecoverable := err.(unrecoverableError); isUnrecoverable {
		return unrecoverable.error
	}

	return err
}

func (r *retrierCore) computeDelay(n uint, err error) time.Duration {
	delayTime := r.delayType(n, err, r)
	if r.maxDelay > 0 && delayTime > r.maxDelay {
		delayTime = r.maxDelay
	}
	return delayTime
}
