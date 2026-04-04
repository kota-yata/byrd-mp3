package header

import (
	"bufio"
	"bytes"
	"testing"
)

func TestReadHeader_NoCRC(t *testing.T) {
	data := []byte{
		0x00, 0x11, // noise before sync
		0xFF, 0xFB, 0x54, 0x00,
	}

	r := bufio.NewReader(bytes.NewReader(data))
	var h MP3FrameHeader
	if err := ReadHeader(&h, r); err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if h.crcValue != [2]byte{} {
		t.Fatalf("crcValue got %v, want zero", h.crcValue)
	}
	if h.HasCRC() {
		t.Fatalf("HasCRC got true, want false")
	}
	if got, free := h.GetBitrateKbps(); free || got != 64 {
		t.Fatalf("bitrate got %d kbps (free=%v), want 64 kbps and free=false", got, free)
	}
	if got := h.GetSampleRate(); got != 48000 {
		t.Fatalf("sample rate got %d, want 48000", got)
	}
	if got := h.Padding(); got {
		t.Fatalf("padding got true, want false")
	}
	if got := h.GetChannelMode(); got != ChannelModeStereo {
		t.Fatalf("channel mode got %s, want %s", got, ChannelModeStereo)
	}
	if got := h.GetModeExtension(); got != 0 {
		t.Fatalf("mode extension got %d, want 0", got)
	}
	if got := h.IsCopyrighted(); got {
		t.Fatalf("copyright got true, want false")
	}
	if got := h.IsOriginal(); got {
		t.Fatalf("original got true, want false")
	}
	if got := h.GetEmphasis(); got != 0 {
		t.Fatalf("emphasis got %d, want 0", got)
	}
	if got, err := h.GetFrameLength(); err != nil || got != 192 {
		t.Fatalf("frame length got (%d, %v), want (192, nil)", got, err)
	}
}

func TestReadHeader_WithCRC(t *testing.T) {
	data := []byte{
		0xFF, 0xFA, 0x52, 0x6D, 0x12, 0x34,
	}

	r := bufio.NewReader(bytes.NewReader(data))
	var h MP3FrameHeader
	if err := ReadHeader(&h, r); err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if !h.HasCRC() {
		t.Fatalf("HasCRC got false, want true")
	}
	if h.crcValue != [2]byte{0x12, 0x34} {
		t.Fatalf("crcValue got %v, want [18 52]", h.crcValue)
	}
	if !bytes.Equal(h.crcTarget, []byte{0x52, 0x6D}) {
		t.Fatalf("crcTarget got %v, want [82 109]", h.crcTarget)
	}
	if got, free := h.GetBitrateKbps(); free || got != 64 {
		t.Fatalf("bitrate got %d kbps (free=%v), want 64 kbps and free=false", got, free)
	}
	if got := h.GetSampleRate(); got != 44100 {
		t.Fatalf("sample rate got %d, want 44100", got)
	}
	if got := h.Padding(); !got {
		t.Fatalf("padding got false, want true")
	}
	if got := h.GetChannelMode(); got != ChannelModeJointStereo {
		t.Fatalf("channel mode got %s, want %s", got, ChannelModeJointStereo)
	}
	if got := h.GetModeExtension(); got != 2 {
		t.Fatalf("mode extension got %d, want 2", got)
	}
	if got := h.IsCopyrighted(); !got {
		t.Fatalf("copyright got false, want true")
	}
	if got := h.IsOriginal(); !got {
		t.Fatalf("original got false, want true")
	}
	if got := h.GetEmphasis(); got != 1 {
		t.Fatalf("emphasis got %d, want 1", got)
	}
	if got, err := h.GetFrameLength(); err != nil || got != 209 {
		t.Fatalf("frame length got (%d, %v), want (209, nil)", got, err)
	}
}

func TestReadHeader_ResyncAfterInvalidCandidate(t *testing.T) {
	data := []byte{
		0xFF, 0xE3, // sync-looking, but unsupported MPEG version candidate
		0xFF, 0xFB, 0x54, 0x00,
	}

	r := bufio.NewReader(bytes.NewReader(data))
	var h MP3FrameHeader
	if err := ReadHeader(&h, r); err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if h.HasCRC() {
		t.Fatalf("HasCRC got true, want false")
	}
	if got, free := h.GetBitrateKbps(); free || got != 64 {
		t.Fatalf("bitrate got %d kbps (free=%v), want 64 kbps and free=false", got, free)
	}
	if got := h.GetSampleRate(); got != 48000 {
		t.Fatalf("sample rate got %d, want 48000", got)
	}
	if got := h.GetChannelMode(); got != ChannelModeStereo {
		t.Fatalf("channel mode got %s, want %s", got, ChannelModeStereo)
	}
}
