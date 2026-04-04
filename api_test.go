package byrd

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeMP3File(t *testing.T) {
	pcm, err := DecodeMP3File(filepath.Join("static", "440hz.mp3"))
	if err != nil {
		t.Fatalf("DecodeMP3File failed: %v", err)
	}
	if pcm == nil {
		t.Fatalf("DecodeMP3File returned nil pcm")
	}
	if pcm.SampleRate == 0 {
		t.Fatalf("sample rate must be non-zero")
	}
	if pcm.Channels <= 0 {
		t.Fatalf("channels must be positive")
	}
	if len(pcm.Samples) == 0 {
		t.Fatalf("samples must be non-empty")
	}
}

func TestConvertMP3FileToWAV(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.wav")
	if err := ConvertMP3FileToWAV(filepath.Join("static", "440hz.mp3"), dst); err != nil {
		t.Fatalf("ConvertMP3FileToWAV failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read wav output: %v", err)
	}
	if len(data) < 44 {
		t.Fatalf("wav output too short: %d", len(data))
	}
	if string(data[0:4]) != "RIFF" {
		t.Fatalf("missing RIFF header: %q", data[0:4])
	}
	if string(data[8:12]) != "WAVE" {
		t.Fatalf("missing WAVE header: %q", data[8:12])
	}
	if string(data[12:16]) != "fmt " {
		t.Fatalf("missing fmt chunk: %q", data[12:16])
	}
	if string(data[36:40]) != "data" {
		t.Fatalf("missing data chunk: %q", data[36:40])
	}

	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	channels := binary.LittleEndian.Uint16(data[22:24])
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	dataSize := binary.LittleEndian.Uint32(data[40:44])

	if audioFormat != 1 {
		t.Fatalf("unexpected audio format: %d", audioFormat)
	}
	if channels == 0 {
		t.Fatalf("unexpected channels: %d", channels)
	}
	if sampleRate == 0 {
		t.Fatalf("unexpected sample rate: %d", sampleRate)
	}
	if bitsPerSample != 16 {
		t.Fatalf("unexpected bits per sample: %d", bitsPerSample)
	}
	if dataSize == 0 {
		t.Fatalf("unexpected data size: %d", dataSize)
	}
}
