//go:build go1.18

package errors

// Must is a generic helper, like template.Must, that wraps a call to a function returning (T, error)
// and panics if the error is non-nil.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
