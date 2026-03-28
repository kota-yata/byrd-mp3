package byrd

import (
	"bufio"
	"testing"
)

func TestReadFirstFrameFromOutputMP3(t *testing.T) {
	f, err := OpenMP3File("output.mp3")
	if err != nil {
		t.Fatalf("failed to open output.mp3: %v", err)
	}
	defer f.Close()

	r := bufio.NewReader(f)

	var h MP3FrameHeader
	if err := ReadHeader(&h, r); err != nil {
		t.Fatalf("failed to read first MP3 frame header: %v", err)
	}

	// Expect MPEG1 Layer III frame with known parameters from header 0xFF FB 54 00
    if got, free := h.GetBitrateKbps(); free || got != 64 {
        t.Fatalf("unexpected bitrate: got %d kbps (free=%v), want 64 kbps", got, free)
    }

    if sr := h.GetSampleRate(); sr != 48000 {
        t.Fatalf("unexpected sample rate: got %d, want 48000", sr)
    }

    if pad := h.Padding(); pad {
        t.Fatalf("unexpected padding bit: got %v, want false", pad)
    }

    if !h.ValidateCRC(r) {
        t.Fatalf("CRC validation failed (expected true or no CRC)")
    }

    if l, err := h.GetFrameLength(); err != nil || l != 192 {
        t.Fatalf("unexpected frame length: got (%d, err=%v), want 192", l, err)
    }
}
