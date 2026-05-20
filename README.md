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

pcm, err := io.ReadAll(dec)
if err != nil {
	log.Fatal(err)
}

log.Printf("decoded %d PCM bytes at %d Hz with %d channels", len(pcm), dec.SampleRate(), dec.Channels())
```

Decode MP3 file at once (non-streaming use cases)

```go
pcmData, err := dec.BatchDecode()
if err != nil {
	log.Fatal(err)
}

if err := pcmData.WriteWAVFile("output.wav"); err != nil {
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
BenchmarkDecode/byrd/440hz-8                   2         761768375 ns/op           1.05 MB/s    73397840 B/op      13511 allocs/op
BenchmarkDecode/go-mp3/440hz-8                 1        1122291417 ns/op           0.71 MB/s    401718416 B/op    832092 allocs/op
BenchmarkDecode/byrd/alarm-8                   4         301212552 ns/op           2.44 MB/s    17954744 B/op       4525 allocs/op
BenchmarkDecode/go-mp3/alarm-8                 3         372493167 ns/op           1.97 MB/s    107437600 B/op    225284 allocs/op
BenchmarkDecode/byrd/song-8                    1        1211818375 ns/op           3.42 MB/s    59609496 B/op      14920 allocs/op
BenchmarkDecode/go-mp3/song-8                  1        1326870209 ns/op           3.12 MB/s    337331008 B/op    693346 allocs/op
BenchmarkDecode/byrd/synth-8                   8         136756088 ns/op           2.25 MB/s     7939654 B/op       1904 allocs/op
BenchmarkDecode/go-mp3/synth-8                 7         166037327 ns/op           1.85 MB/s    48806340 B/op     102876 allocs/op
BenchmarkDecode/byrd/circle-reading-8                  1        41701810375 ns/op          2.04 MB/s    2376429496 B/op   618374 allocs/op
BenchmarkDecode/go-mp3/circle-reading-8                1        50237129917 ns/op          1.69 MB/s    14913808680 B/op        31594036 allocs/op
```

```
goos: linux
goarch: amd64
pkg: byrd-bench
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkDecode/byrd/440hz-2                   1        1386793842 ns/op           0.58 MB/s    140145296 B/op      7535 allocs/op
BenchmarkDecode/go-mp3/440hz-2                 1        1895197625 ns/op           0.42 MB/s    401718176 B/op    832089 allocs/op
BenchmarkDecode/byrd/alarm-2                   2         557039262 ns/op           1.32 MB/s    46667136 B/op       2912 allocs/op
BenchmarkDecode/go-mp3/alarm-2                 2         676362464 ns/op           1.08 MB/s    107437632 B/op    225285 allocs/op
BenchmarkDecode/byrd/song-2                    1        2168383865 ns/op           1.91 MB/s    145648976 B/op      9967 allocs/op
BenchmarkDecode/go-mp3/song-2                  1        2297369434 ns/op           1.80 MB/s    337330800 B/op    693343 allocs/op
BenchmarkDecode/byrd/synth-2                   5         269016935 ns/op           1.14 MB/s    18706784 B/op       1164 allocs/op
BenchmarkDecode/go-mp3/synth-2                 4         322580975 ns/op           0.95 MB/s    48806280 B/op     102876 allocs/op
BenchmarkDecode/byrd/circle-reading-2                  1        77254737684 ns/op          1.10 MB/s    6377618904 B/op   392294 allocs/op
BenchmarkDecode/go-mp3/circle-reading-2                1        89520887751 ns/op          0.95 MB/s    14913806568 B/op        31594022 allocs/op
```

In comparison to hajimehoshi/go-mp3, allocs/op for Byrd is very few for all sample files. It's also faster looking at the ns/op and throughput
