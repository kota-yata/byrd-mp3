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

### Benchmark result as of v0.1.0 with go-mp3

```
goos: darwin
goarch: arm64
pkg: byrd-bench
cpu: Apple M2
BenchmarkDecode/byrd/440hz-8                   2         745993584 ns/op          1.07 MB/s     140336464 B/op     13509 allocs/op
BenchmarkDecode/go-mp3/440hz-8                 1        1144709833 ns/op          0.70 MB/s     401723832 B/op    832098 allocs/op
BenchmarkDecode/byrd/alarm-8                   4         311987406 ns/op          2.35 MB/s     46718656 B/op       4522 allocs/op
BenchmarkDecode/go-mp3/alarm-8                 3         372543819 ns/op          1.97 MB/s     107437696 B/op    225285 allocs/op
BenchmarkDecode/byrd/song-8                    1        1194510209 ns/op          3.47 MB/s     145807408 B/op     14918 allocs/op
BenchmarkDecode/go-mp3/song-8                  1        1326219416 ns/op          3.12 MB/s     337330736 B/op    693343 allocs/op
BenchmarkDecode/byrd/synth-8                   8         135744922 ns/op          2.26 MB/s     18730332 B/op       1899 allocs/op
BenchmarkDecode/go-mp3/synth-8                 7         165773417 ns/op          1.85 MB/s     48806358 B/op     102876 allocs/op
```

```
goos: linux
goarch: amd64
pkg: byrd-bench
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkDecode/byrd/440hz-2                   1        1460677840 ns/op           0.55 MB/s    140336464 B/op     13509 allocs/op
BenchmarkDecode/go-mp3/440hz-2                 1        1662379093 ns/op           0.48 MB/s    401718032 B/op    832088 allocs/op
BenchmarkDecode/byrd/alarm-2                   2         540961294 ns/op           1.36 MB/s    46718656 B/op       4522 allocs/op
BenchmarkDecode/go-mp3/alarm-2                 2         626373076 ns/op           1.17 MB/s    107437504 B/op    225284 allocs/op
BenchmarkDecode/byrd/song-2                    1        2047548510 ns/op           2.02 MB/s    145807408 B/op     14918 allocs/op
BenchmarkDecode/go-mp3/song-2                  1        2472060620 ns/op           1.68 MB/s    337330784 B/op    693343 allocs/op
BenchmarkDecode/byrd/synth-2                   4         257663386 ns/op           1.19 MB/s    18730304 B/op       1899 allocs/op
BenchmarkDecode/go-mp3/synth-2                 4         253750626 ns/op           1.21 MB/s    48806304 B/op     102876 allocs/op
```

In comparison to hajimehoshi/go-mp3, allocs/op for Byrd is very few for all sample files. It's also faster looking at the ns/op and throughput
