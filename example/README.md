# Examples

`byrd` exposes two main ways to use the decoder:

- Stream decoded PCM with `NewDecoder`
- Convert an MP3 file to a WAV file with `ConvertMP3FileToWAV`

## 1. Stream PCM bytes

This example decodes an MP3 and prints the sample rate and decoded PCM size.

```bash
go run ./example/stream_pcm static/440hz.mp3
```

The output PCM format is always:

- 16-bit signed little-endian
- stereo

## 2. Convert MP3 to WAV

This example decodes an MP3 and writes a `.wav` file.

```bash
go run ./example/to_wav static/440hz.mp3 /tmp/440hz.wav
```

## Notes

- Run these commands from the repository root.
- `NewDecoder` eagerly decodes the full MP3 and then serves PCM bytes via `Read`.
- `DecodeMP3File` returns raw samples as `PCMData`.
- `ConvertMP3FileToWAV` is the simplest way to produce a playable WAV file.
