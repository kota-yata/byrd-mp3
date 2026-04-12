package decoder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMP3FileAcceptedExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "sample.mp3"),
		filepath.Join(dir, "sample.MP3"),
	}

	for _, path := range paths {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", filepath.Base(path), err)
		}

		f, err := OpenMP3File(path)
		if err != nil {
			t.Fatalf("OpenMP3File(%q) returned error: %v", filepath.Base(path), err)
		}
		f.Close()
	}
}

func TestOpenMP3FileRejectedExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.wav")
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create %s: %v", filepath.Base(path), err)
	}

	f, err := OpenMP3File(path)
	if err == nil {
		f.Close()
		t.Fatalf("OpenMP3File(%q) unexpectedly succeeded", filepath.Base(path))
	}
}
