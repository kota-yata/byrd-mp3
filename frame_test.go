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

	var h MP3Frame
	if err := ReadHeader(&h, r); err != nil {
		t.Fatalf("failed to read first MP3 frame header: %v", err)
	}

	// Expect MPEG1 Layer III frame with known parameters from header 0xFF FB 54 00
	if got, free := GetBitrateKbps(&h); free || got != 64 {
		t.Fatalf("unexpected bitrate: got %d kbps (free=%v), want 64 kbps", got, free)
	}

	if sr := GetSampleRate(&h); sr != 48000 {
		t.Fatalf("unexpected sample rate: got %d, want 48000", sr)
	}

	if pad := GetPadding(&h); pad != 0 {
		t.Fatalf("unexpected padding bit: got %d, want 0", pad)
	}

	if !ValidateCRC(&h, r) {
		t.Fatalf("CRC validation failed (expected true or no CRC)")
	}

	if l, err := GetFrameLength(&h); err != nil || l != 192 {
		t.Fatalf("unexpected frame length: got (%d, err=%v), want 192", l, err)
	}

	if !isStereo(&h) {
		t.Fatalf("expected stereo channel mode")
	}
}
