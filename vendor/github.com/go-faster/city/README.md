# city [![](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/city#section-documentation) [![](https://img.shields.io/codecov/c/github/go-faster/city?label=cover)](https://codecov.io/gh/go-faster/city) [![stable](https://img.shields.io/badge/-stable-brightgreen)](https://go-faster.org/docs/projects/status#stable)
[CityHash](https://github.com/google/cityhash) in Go. Fork of [tenfyzhong/cityhash](https://github.com/tenfyzhong/cityhash).

Note: **prefer [xxhash](https://github.com/cespare/xxhash) as non-cryptographic hash algorithm**, this package is intended 
for places where CityHash is already used.

CityHash **is not compatible** to [FarmHash](https://github.com/google/farmhash), use [go-farm](https://github.com/dgryski/go-farm).

```console
go get github.com/go-faster/city
```

```go
city.Hash128([]byte("hello"))
```

* Faster
* Supports ClickHouse hash

```
name            old time/op    new time/op    delta
CityHash64-32      333ns ± 2%     108ns ± 3%   -67.57%  (p=0.000 n=10+10)
CityHash128-32     347ns ± 2%     112ns ± 2%   -67.74%  (p=0.000 n=9+10)

name            old speed      new speed      delta
CityHash64-32   3.08GB/s ± 2%  9.49GB/s ± 3%  +208.40%  (p=0.000 n=10+10)
CityHash128-32  2.95GB/s ± 2%  9.14GB/s ± 2%  +209.98%  (p=0.000 n=9+10)
```

## Benchmarks
```
goos: linux
goarch: amd64
pkg: github.com/go-faster/city
cpu: AMD Ryzen 9 5950X 16-Core Processor
BenchmarkClickHouse128/16     2213.98 MB/s
BenchmarkClickHouse128/64     4712.24 MB/s
BenchmarkClickHouse128/256    7561.58 MB/s
BenchmarkClickHouse128/1024  10158.98 MB/s
BenchmarkClickHouse64        10379.89 MB/s
BenchmarkCityHash32           3140.54 MB/s
BenchmarkCityHash64           9508.45 MB/s
BenchmarkCityHash128          9304.27 MB/s
BenchmarkCityHash64Small      2700.84 MB/s
BenchmarkCityHash128Small     1175.65 MB/s
```
