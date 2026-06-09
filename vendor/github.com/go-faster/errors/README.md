# errors [![Go Reference](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/errors#section-documentation) [![codecov](https://img.shields.io/codecov/c/github/go-faster/errors?label=cover)](https://codecov.io/gh/go-faster/errors)

Fork of [xerrors](https://pkg.go.dev/golang.org/x/xerrors) with explicit [Wrap](https://pkg.go.dev/github.com/go-faster/errors#Wrap) instead of `%w`.

> Clear is better than clever.

```
go get github.com/go-faster/errors
```

```go
errors.Wrap(err, "message")
```

## Why
* Using `Wrap` is the most explicit way to wrap errors
* Wrapping with `fmt.Errorf("foo: %w", err)` is implicit, redundant and error-prone
* Parsing `"foo: %w"` is implicit, redundant and slow
* The [pkg/errors](https://github.com/pkg/errors) and [xerrors](https://pkg.go.dev/golang.org/x/xerrors) are not maintainted
* The [cockroachdb/errors](https://github.com/cockroachdb/errors) is too big
* The `errors` has no caller stack trace

## Don't need traces?
Call `errors.DisableTrace` or use build tag `noerrtrace`.

## Additional features

### Into

Generic type assertion for errors.

```go
// Into finds the first error in err's chain that matches target type T, and if so, returns it.
//
// Into is type-safe alternative to As.
func Into[T error](err error) (val T, ok bool)
```

```go
if pathError, ok := errors.Into[*os.PathError](err); ok {
    fmt.Println("Failed at path:", pathError.Path)
}
```

### Must

Must is a generic helper, like template.Must, that wraps a call to a function returning (T, error)
and panics if the error is non-nil.

```go
func Must[T any](val T, err error) T
```

## License

BSD-3-Clause, same as Go sources
