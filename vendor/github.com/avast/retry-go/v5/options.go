package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Function signature of retry if function
type RetryIfFunc func(error) bool

// Function signature of OnRetry function
type OnRetryFunc func(attempt uint, err error)

// DelayContext provides configuration values needed for delay calculation.
type DelayContext interface {
	Delay() time.Duration
	MaxJitter() time.Duration
	MaxBackOffN() uint
	MaxDelay() time.Duration
}

// DelayTypeFunc is called to return the next delay to wait after the retriable function fails on `err` after `n` attempts.
type DelayTypeFunc func(n uint, err error, config DelayContext) time.Duration

// Timer represents the timer used to track time for a retry.
type Timer interface {
	After(time.Duration) <-chan time.Time
}

// retrierCore holds the core configuration and business logic for retry operations.
// this is then used by Retrier and RetrierWithData public APIs
type retrierCore struct {
	attempts                      uint
	attemptsForError              map[error]uint
	delay                         time.Duration
	maxDelay                      time.Duration
	maxJitter                     time.Duration
	onRetry                       OnRetryFunc
	retryIf                       RetryIfFunc
	delayType                     DelayTypeFunc
	lastErrorOnly                 bool
	context                       context.Context
	timer                         Timer
	wrapContextErrorWithLastError bool

	maxBackOffN uint // pre-computed for BackOffDelay, immutable after New()
}

// Delay implements DelayContext
func (r *retrierCore) Delay() time.Duration {
	return r.delay
}

// MaxJitter implements DelayContext
func (r *retrierCore) MaxJitter() time.Duration {
	return r.maxJitter
}

// MaxBackOffN implements DelayContext
func (r *retrierCore) MaxBackOffN() uint {
	return r.maxBackOffN
}

// MaxDelay implements DelayContext
func (r *retrierCore) MaxDelay() time.Duration {
	return r.maxDelay
}

// Retrier is for retry operations that return only an error.
type Retrier struct {
	*retrierCore
}

// RetrierWithData is for retry operations that return data and an error.
type RetrierWithData[T any] struct {
	*retrierCore
}

// Option represents an option for retry.
type Option func(*retrierCore)

func newRetrieerCore(opts ...Option) *retrierCore {
	core := &retrierCore{
		attempts:         uint(10),
		attemptsForError: make(map[error]uint),
		delay:            100 * time.Millisecond,
		maxJitter:        100 * time.Millisecond,
		onRetry:          func(n uint, err error) {},
		retryIf:          IsRecoverable,
		delayType:        CombineDelay(BackOffDelay, RandomDelay),
		lastErrorOnly:    false,
		context:          context.Background(),
		timer:            &timerImpl{},
	}

	for _, opt := range opts {
		opt(core)
	}

	const maxBackOffN uint = 62
	core.maxBackOffN = maxBackOffN
	if core.delay < 0 {
		core.delay = 0
	} else if core.delay > 0 {
		core.maxBackOffN = maxBackOffN - uint(math.Floor(math.Log2(float64(core.delay))))
	}

	return core
}

// New creates a new Retrier with the given options.
// The returned Retrier can be safely reused across multiple retry operations.
func New(opts ...Option) *Retrier {
	return &Retrier{retrierCore: newRetrieerCore(opts...)}
}

// NewWithData creates a new RetrierWithData[T] with the given options.
// The returned retrier can be safely reused across multiple retry operations.
func NewWithData[T any](opts ...Option) *RetrierWithData[T] {
	return &RetrierWithData[T]{retrierCore: newRetrieerCore(opts...)}
}

func emptyOption(r *retrierCore) {}

// return the direct last error that came from the retried function
// default is false (return wrapped errors with everything)
func LastErrorOnly(lastErrorOnly bool) Option {
	return func(r *retrierCore) {
		r.lastErrorOnly = lastErrorOnly
	}
}

// Attempts set count of retry. Setting to 0 will retry until the retried function succeeds.
// default is 10
func Attempts(attempts uint) Option {
	return func(r *retrierCore) {
		r.attempts = attempts
	}
}

// UntilSucceeded will retry until the retried function succeeds. Equivalent to setting Attempts(0).
func UntilSucceeded() Option {
	return func(r *retrierCore) {
		r.attempts = 0
	}
}

// AttemptsForError sets count of retry in case execution results in given `err`
// Retries for the given `err` are also counted against total retries.
// The retry will stop if any of given retries is exhausted.
//
// added in 4.3.0
func AttemptsForError(attempts uint, err error) Option {
	return func(r *retrierCore) {
		r.attemptsForError[err] = attempts
	}
}

// Delay set delay between retry
// default is 100ms
func Delay(delay time.Duration) Option {
	return func(r *retrierCore) {
		r.delay = delay
	}
}

// MaxDelay set maximum delay between retry
// does not apply by default
func MaxDelay(maxDelay time.Duration) Option {
	return func(r *retrierCore) {
		r.maxDelay = maxDelay
	}
}

// MaxJitter sets the maximum random Jitter between retries for RandomDelay
func MaxJitter(maxJitter time.Duration) Option {
	return func(r *retrierCore) {
		r.maxJitter = maxJitter
	}
}

// DelayType set type of the delay between retries
// default is a combination of BackOffDelay and RandomDelay for exponential backoff with jitter
func DelayType(delayType DelayTypeFunc) Option {
	if delayType == nil {
		return emptyOption
	}
	return func(r *retrierCore) {
		r.delayType = delayType
	}
}

// BackOffDelay is a DelayType which increases delay between consecutive retries
func BackOffDelay(n uint, _ error, config DelayContext) time.Duration {
	maxBackOffN := config.MaxBackOffN()
	n--
	if n > maxBackOffN {
		n = maxBackOffN
	}
	return config.Delay() << n
}

// FixedDelay is a DelayType which keeps delay the same through all iterations
func FixedDelay(_ uint, _ error, config DelayContext) time.Duration {
	return config.Delay()
}

// RandomDelay is a DelayType which picks a random delay up to maxJitter
func RandomDelay(_ uint, _ error, config DelayContext) time.Duration {
	maxJitter := config.MaxJitter()
	if maxJitter == 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(maxJitter)))
}

// CombineDelay is a DelayType the combines all of the specified delays into a new DelayTypeFunc
func CombineDelay(delays ...DelayTypeFunc) DelayTypeFunc {
	const maxInt64 = uint64(math.MaxInt64)

	return func(n uint, err error, config DelayContext) time.Duration {
		var total uint64
		for _, delay := range delays {
			total += uint64(delay(n, err, config))
			if total > maxInt64 {
				total = maxInt64
			}
		}

		return time.Duration(total)
	}
}

// FullJitterBackoffDelay is a DelayTypeFunc that calculates delay using exponential backoff
// with full jitter. The delay is a random value between 0 and the current backoff ceiling.
// Formula: sleep = random_between(0, min(cap, base * 2^attempt))
// It uses config.Delay as the base delay and config.MaxDelay as the cap.
func FullJitterBackoffDelay(n uint, err error, config DelayContext) time.Duration {
	// Calculate the exponential backoff ceiling for the current attempt
	backoffCeiling := float64(config.Delay()) * math.Pow(2, float64(n))
	currentCap := float64(config.MaxDelay())

	// If MaxDelay is set and backoffCeiling exceeds it, cap at MaxDelay
	if currentCap > 0 && backoffCeiling > currentCap {
		backoffCeiling = currentCap
	}

	// Ensure backoffCeiling is at least 0
	if backoffCeiling < 0 {
		backoffCeiling = 0
	}

	// Add jitter: random value between 0 and backoffCeiling
	// rand.Int63n panics if argument is <= 0
	if backoffCeiling <= 0 {
		return 0 // No delay if ceiling is zero or negative
	}

	jitter := rand.Int63n(int64(backoffCeiling)) // #nosec G404 -- Using math/rand is acceptable for non-security critical jitter.
	return time.Duration(jitter)
}

// OnRetry function callback are called each retry
//
// log each retry example:
//
//	retry.New().Do(
//		func() error {
//			return errors.New("some error")
//		},
//		retry.OnRetry(func(n uint, err error) {
//			log.Printf("#%d: %s\n", n, err)
//		}),
//	)
func OnRetry(onRetry OnRetryFunc) Option {
	if onRetry == nil {
		return emptyOption
	}
	return func(r *retrierCore) {
		r.onRetry = onRetry
	}
}

// RetryIf controls whether a retry should be attempted after an error
// (assuming there are any retry attempts remaining)
//
// skip retry if special error example:
//
//	retry.New().Do(
//		func() error {
//			return errors.New("special error")
//		},
//		retry.RetryIf(func(err error) bool {
//			if err.Error() == "special error" {
//				return false
//			}
//			return true
//		})
//	)
//
// By default RetryIf stops execution if the error is wrapped using `retry.Unrecoverable`,
// so above example may also be shortened to:
//
//	retry.New().Do(
//		func() error {
//			return retry.Unrecoverable(errors.New("special error"))
//		}
//	)
func RetryIf(retryIf RetryIfFunc) Option {
	if retryIf == nil {
		return emptyOption
	}
	return func(r *retrierCore) {
		r.retryIf = retryIf
	}
}

// Context allow to set context of retry
// default are Background context
//
// example of immediately cancellation (maybe it isn't the best example, but it describes behavior enough; I hope)
//
//	ctx, cancel := context.WithCancel(context.Background())
//	cancel()
//
//	retry.New().Do(
//		func() error {
//			...
//		},
//		retry.Context(ctx),
//	)
func Context(ctx context.Context) Option {
	return func(r *retrierCore) {
		r.context = ctx
	}
}

// WithTimer provides a way to swap out timer module implementations.
// This primarily is useful for mocking/testing, where you may not want to explicitly wait for a set duration
// for retries.
//
// example of augmenting time.After with a print statement
//
//	type struct MyTimer {}
//
//	func (t *MyTimer) After(d time.Duration) <- chan time.Time {
//	    fmt.Print("Timer called!")
//	    return time.After(d)
//	}
//
//	retry.New().Do(
//	    func() error { ... },
//		   retry.WithTimer(&MyTimer{})
//	)
func WithTimer(t Timer) Option {
	return func(r *retrierCore) {
		r.timer = t
	}
}

// WrapContextErrorWithLastError allows the context error to be returned wrapped with the last error that the
// retried function returned. This is only applicable when Attempts is set to 0 to retry indefinitly and when
// using a context to cancel / timeout
//
// default is false
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	retry.New().Do(
//		func() error {
//			...
//		},
//		retry.Context(ctx),
//		retry.Attempts(0),
//		retry.WrapContextErrorWithLastError(true),
//	)
func WrapContextErrorWithLastError(wrapContextErrorWithLastError bool) Option {
	return func(r *retrierCore) {
		r.wrapContextErrorWithLastError = wrapContextErrorWithLastError
	}
}
