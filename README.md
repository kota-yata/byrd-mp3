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

### Benchmark result as of v0.1.1 with go-mp3

```
goos: darwin
goarch: arm64
pkg: byrd-bench
cpu: Apple M2
BenchmarkDecode/byrd/440hz-8                   2         772611416 ns/op           1.03 MB/s    140150616 B/op      7541 allocs/op
BenchmarkDecode/go-mp3/440hz-8                 1        1162266375 ns/op           0.69 MB/s    401718448 B/op    832091 allocs/op
BenchmarkDecode/byrd/alarm-8                   4         299168010 ns/op           2.45 MB/s    46667164 B/op       2912 allocs/op
BenchmarkDecode/go-mp3/alarm-8                 3         372317639 ns/op           1.97 MB/s    107437728 B/op    225285 allocs/op
BenchmarkDecode/byrd/song-8                    1        1205008584 ns/op           3.44 MB/s    145648976 B/op      9967 allocs/op
BenchmarkDecode/go-mp3/song-8                  1        1365143459 ns/op           3.04 MB/s    337331008 B/op    693346 allocs/op
BenchmarkDecode/byrd/synth-8                   8         136862448 ns/op           2.25 MB/s    18706784 B/op       1164 allocs/op
BenchmarkDecode/go-mp3/synth-8                 7         167811369 ns/op           1.83 MB/s    48806390 B/op     102876 allocs/op
BenchmarkDecode/byrd/circle-reading-8                  1        41794333000 ns/op          2.03 MB/s    6377618920 B/op   392295 allocs/op
BenchmarkDecode/go-mp3/circle-reading-8                1        50479988500 ns/op          1.68 MB/s    14913809400 B/op        31594042 allocs/op
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
