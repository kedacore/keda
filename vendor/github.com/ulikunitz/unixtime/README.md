# Package unixtime

This package has been created to provide functions to convert between Go
time values and integers representing the Unix time in micro- or
milliseconds.

Ian Lance Taylor suggested to create such a package in the discussion of
Go issue [#27782](https://github.com/golang/go/issues/27782). The
package should be used to measure, whether there is really a need for
this functionality that would justify the inclusion in the standard
library. Note that the issue was a duplicate of
[#18935](https://github.com/golang/go/issues/18935).

## Installing the package

Use the get get command to download the module.

```bash
$ go get -u github.com/ulikunitz/unixtime
```

## Using the package

```go
import (
        "fmt"
        "log"
        "time"

        "github.com/ulikunitz/unixtime"
)

func Example() {
        t, err := time.Parse(time.RFC3339Nano, "1961-04-12T09:06:59.7+03:00")
        if err != nil {
                log.Fatalf("Parse error %s", err)
        }

        ms := unixtime.Milli(t)
        fmt.Printf("Unix time: %d ms\n", ms)

        tms := unixtime.FromMilli(ms)
        fmt.Printf("FromMilli: %s\n", tms.Format(time.RFC3339Nano))

        µs := unixtime.Micro(t)
        fmt.Printf("Unix time: %d µs\n", µs)

        tµs := unixtime.FromMicro(µs)
        fmt.Printf("FromMicro: %s\n", tµs.Format(time.RFC3339Nano))
}
```
