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

```go
f, err := os.Open("input.mp3")
if err != nil {
	log.Fatal(err)
}
defer f.Close()

dec, err := byrd.NewDecoder(f)
if err != nil {
	log.Fatal(err)
}

pcm, err := dec.Decode()
if err != nil {
	log.Fatal(err)
}

if err := pcm.WriteWAVFile("output.wav"); err != nil {
	log.Fatal(err)
}
```

See examples under example/ for further usage.

### Benchmark result as of v0.0.1 with go-mp3

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

```
goos: linux
goarch: amd64
pkg: byrd-bench
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkDecode/byrd/440hz-2                   1        2299995251 ns/op           0.35 MB/s    140336576 B/op     13510 allocs/op
BenchmarkDecode/go-mp3/440hz-2                 1        3386158802 ns/op           0.24 MB/s    401718688 B/op    832092 allocs/op
BenchmarkDecode/byrd/alarm-2                   2         639494856 ns/op           1.15 MB/s    46718656 B/op       4522 allocs/op
BenchmarkDecode/go-mp3/alarm-2                 2         841921006 ns/op           0.87 MB/s    107440228 B/op    225287 allocs/op
BenchmarkDecode/byrd/song-2                    1        2817046877 ns/op           1.47 MB/s    145803928 B/op     14920 allocs/op
BenchmarkDecode/go-mp3/song-2                  1        2469238580 ns/op           1.68 MB/s    337328448 B/op    693343 allocs/op
BenchmarkDecode/byrd/synth-2                   4         318255502 ns/op           0.97 MB/s    18730304 B/op       1899 allocs/op
BenchmarkDecode/go-mp3/synth-2                 4         280196347 ns/op           1.10 MB/s    48806348 B/op     102876 allocs/op
```

In comparison to hajimehoshi/go-mp3, allocs/op for Byrd is very few for all sample files. It's also faster looking at the ns/op and throughput
