![Gopher with Byrd letter](./static/byrd-gopher.png)
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

### Benchmark result as of v0.2.0 with go-mp3

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
BenchmarkDecode/byrd/440hz-2                   1        1414838350 ns/op           0.56 MB/s    73397944 B/op   13512 allocs/op
BenchmarkDecode/go-mp3/440hz-2                 1        1789270359 ns/op           0.45 MB/s    401718432 B/op  832091 allocs/op
BenchmarkDecode/byrd/alarm-2                   2         563254048 ns/op           1.30 MB/s    17954744 B/op    4525 allocs/op
BenchmarkDecode/go-mp3/alarm-2                 2         583913758 ns/op           1.26 MB/s    107437704 B/op  225285 allocs/op
BenchmarkDecode/byrd/song-2                    1        2236068400 ns/op           1.85 MB/s    59609384 B/op   14919 allocs/op
BenchmarkDecode/go-mp3/song-2                  1        2495641055 ns/op           1.66 MB/s    337330784 B/op  693343 allocs/op
BenchmarkDecode/byrd/synth-2                   5         239184430 ns/op           1.29 MB/s     7939640 B/op    1904 allocs/op
BenchmarkDecode/go-mp3/synth-2                 4         265875043 ns/op           1.16 MB/s    48806272 B/op  102876 allocs/op
BenchmarkDecode/byrd/circle-reading-2                  1        83411402901 ns/op          1.02 MB/s    2376424176 B/op         618368 allocs/op
BenchmarkDecode/go-mp3/circle-reading-2                1        82844266854 ns/op          1.03 MB/s    14913806760 B/op      31594024 allocs/op
```

Byrd decodes mp3 with lower allocation bytes per op
