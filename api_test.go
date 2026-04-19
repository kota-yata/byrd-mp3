package byrd

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func mustListStaticMP3Paths(t *testing.T) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("static", "*.mp3"))
	if err != nil {
		t.Fatalf("failed to list static mp3 files: %v", err)
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		t.Fatalf("no mp3 files found under static/")
	}
	return paths
}

func mustDecodePath(t *testing.T, path string) *PCMData {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open mp3: %v", err)
	}
	defer f.Close()

	dec, err := NewDecoder(f)
	if err != nil {
		t.Fatalf("NewDecoder failed: %v", err)
	}

	pcm, err := dec.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	return pcm
}

func TestDecoder_Decode(t *testing.T) {
	for _, path := range mustListStaticMP3Paths(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			pcm := mustDecodePath(t, path)
			if pcm == nil {
				t.Fatalf("Decode returned nil pcm")
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
		})
	}
}

func TestPCMData_WriteWAVFile(t *testing.T) {
	for _, path := range mustListStaticMP3Paths(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			pcm := mustDecodePath(t, path)

			dst := filepath.Join(t.TempDir(), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))+".wav")
			if err := pcm.WriteWAVFile(dst); err != nil {
				t.Fatalf("WriteWAVFile failed: %v", err)
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
		})
	}
}

func TestWriteStaticDecodedWAVFiles(t *testing.T) {
	for _, path := range mustListStaticMP3Paths(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			pcm := mustDecodePath(t, path)

			outPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".decoded.wav"
			if err := pcm.WriteWAVFile(outPath); err != nil {
				t.Fatalf("WriteWAVFile failed for %s: %v", filepath.Base(path), err)
			}

			info, err := os.Stat(outPath)
			if err != nil {
				t.Fatalf("failed to stat %s: %v", filepath.Base(outPath), err)
			}
			if info.Size() <= 44 {
				t.Fatalf("wav output too small: %s size=%d", filepath.Base(outPath), info.Size())
			}
			t.Logf("wrote %s", outPath)
		})
	}
}
