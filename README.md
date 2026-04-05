## Byrd
MP3 (MPEG-1 Layer 3) decoder in Go. No dependency to third-party libraries.

Byrd reduces alloc count per operation by reusing memory addresses from previous frame as much as possible.

### Usage
```bash
go get github.com/kota-yata/byrd-mp3
```
```go
import byrd "github.com/kota-yata/byrd-mp3"
```

Runnable examples are under [example/README.md](/Users/kota-yata/dev/repos/mp3/example/README.md).

### Benchmark result at 2026/04/05 with go-mp3 (archived)

```
goos: darwin
goarch: arm64
pkg: byrd-bench
cpu: Apple M2
BenchmarkDecode/byrd/440hz-8                   2         840817834 ns/op         0.95 MB/s     140336472 B/op     13509 allocs/op
BenchmarkDecode/go-mp3/440hz-8                 1        1180884459 ns/op         0.68 MB/s     401718208 B/op    832089 allocs/op

BenchmarkDecode/byrd/alarm-8                   4         326619854 ns/op         2.25 MB/s     46718712 B/op       4522 allocs/op
BenchmarkDecode/go-mp3/alarm-8                 3         370107861 ns/op         1.98 MB/s     107437808 B/op    225286 allocs/op

BenchmarkDecode/byrd/song-8                    1        1247463417 ns/op         3.32 MB/s     145804824 B/op     14920 allocs/op
BenchmarkDecode/go-mp3/song-8                  1        1330620834 ns/op         3.11 MB/s     337329008 B/op    693348 allocs/op

BenchmarkDecode/byrd/synth-8                   7         142934625 ns/op         2.15 MB/s     18730320 B/op       1899 allocs/op
BenchmarkDecode/go-mp3/synth-8                 7         166538673 ns/op         1.85 MB/s     48806436 B/op     102877 allocs/op
```

In comparison to hajimehoshi/go-mp3, allocs/op for Byrd is very few for all sample files. It's also faster looking at the ns/op and throughput
