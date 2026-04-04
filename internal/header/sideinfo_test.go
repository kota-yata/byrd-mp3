package header

import (
	"bufio"
	"bytes"
	"testing"
)

// bitWriter is a tiny helper to pack bitfields for side info bytes
type bitWriter struct {
	b    []byte
	cur  byte
	nbit int
}

func (w *bitWriter) write(n int, v uint32) {
	for i := n - 1; i >= 0; i-- {
		bit := byte((v >> uint(i)) & 1)
		w.cur = (w.cur << 1) | bit
		w.nbit++
		if w.nbit == 8 {
			w.b = append(w.b, w.cur)
			w.cur = 0
			w.nbit = 0
		}
	}
}

func (w *bitWriter) bytes() []byte {
	if w.nbit != 0 {
		// pad remaining bits with zeros
		for w.nbit != 0 {
			w.cur <<= 1
			w.nbit++
			if w.nbit == 8 {
				w.b = append(w.b, w.cur)
				w.cur = 0
				w.nbit = 0
			}
		}
	}
	return w.b
}

func TestReadSideInfo_Stereo_NonWindowSwitching(t *testing.T) {
	// Prepare a stereo header with no CRC
	var h MP3FrameHeader
	h.flag1 |= 1 << 6 // protection bit = 1 (no CRC)
	h.flag2 = 0       // ChannelModeStereo

	// Build 32-byte side info for stereo without window switching
	var bw bitWriter
	// main_data_begin (9)
	bw.write(9, 0x101)
	// private_bits for stereo (3)
	bw.write(3, 0b101)
	// scfsi: ch0 bands 4 bits, ch1 bands 4 bits
	// ch0: 1,0,1,0; ch1: 0,1,0,1
	for _, v := range []uint32{1, 0, 1, 0} {
		bw.write(1, v)
	}
	for _, v := range []uint32{0, 1, 0, 1} {
		bw.write(1, v)
	}

	// For each of 2 granules and 2 channels, write fields for non-window-switching path
	type gcVals struct {
		p23 uint32
		bv  uint32
		gg  uint32
		sfc uint32
		ts  [3]uint32
		r0  uint32
		r1  uint32
		pre uint32
		sfs uint32
		c1t uint32
	}

	// Use distinct values to verify mapping
	vals := [2][2]gcVals{
		{
			{p23: 0x123, bv: 0x101, gg: 0x55, sfc: 0xA, ts: [3]uint32{1, 2, 3}, r0: 9, r1: 3, pre: 1, sfs: 0, c1t: 1},
			{p23: 0x222, bv: 0x080, gg: 0x40, sfc: 0x3, ts: [3]uint32{4, 5, 6}, r0: 7, r1: 5, pre: 0, sfs: 1, c1t: 0},
		},
		{
			{p23: 0x111, bv: 0x055, gg: 0x7F, sfc: 0xF, ts: [3]uint32{7, 8, 9}, r0: 5, r1: 2, pre: 1, sfs: 1, c1t: 0},
			{p23: 0x00F, bv: 0x001, gg: 0x01, sfc: 0x1, ts: [3]uint32{10, 11, 12}, r0: 3, r1: 6, pre: 0, sfs: 0, c1t: 1},
		},
	}

	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < 2; ch++ {
			v := vals[gr][ch]
			bw.write(12, v.p23)
			bw.write(9, v.bv)
			bw.write(8, v.gg)
			bw.write(4, v.sfc)
			bw.write(1, 0) // window_switching = 0
			for i := 0; i < 3; i++ {
				bw.write(5, v.ts[i])
			}
			bw.write(4, v.r0)
			bw.write(3, v.r1)
			bw.write(1, v.pre)
			bw.write(1, v.sfs)
			bw.write(1, v.c1t)
		}
	}

	data := bw.bytes()
	if len(data) != 32 {
		t.Fatalf("built side info length = %d, want 32", len(data))
	}

	r := bufio.NewReader(bytes.NewReader(data))
	si, err := ReadSideInfo(&h, r, GetSideInfoLength(&h))
	if err != nil {
		t.Fatalf("ReadSideInfo failed: %v", err)
	}

	if si.MainDataBegin != 0x101 {
		t.Fatalf("MainDataBegin got %d, want 0x101", si.MainDataBegin)
	}
	if got := si.SCFSI[0]; got != [4]byte{1, 0, 1, 0} {
		t.Fatalf("SCFSI ch0 got %v, want [1 0 1 0]", got)
	}
	if got := si.SCFSI[1]; got != [4]byte{0, 1, 0, 1} {
		t.Fatalf("SCFSI ch1 got %v, want [0 1 0 1]", got)
	}

	// spot check a few fields
	g := si.Granule[0][0]
	if g.Part23Length != uint16(vals[0][0].p23) || g.BigValues != uint16(vals[0][0].bv) || g.GlobalGain != byte(vals[0][0].gg) || g.ScalefacCompress != byte(vals[0][0].sfc) {
		t.Fatalf("granule[0][0] header fields mismatch: %+v", g)
	}
	if g.GetWindowSwitching() {
		t.Fatalf("granule[0][0] unexpected window switching true")
	}
	if g.TableSelect != [3]byte{1, 2, 3} {
		t.Fatalf("granule[0][0] TableSelect got %v, want [1 2 3]", g.TableSelect)
	}
	if g.Region0Count != byte(vals[0][0].r0) || g.Region1Count != byte(vals[0][0].r1) {
		t.Fatalf("granule[0][0] region counts got (%d,%d)", g.Region0Count, g.Region1Count)
	}
	if !g.GetPreflag() || g.GetScalefacScale() || !g.GetCount1TableSelect() {
		t.Fatalf("granule[0][0] flags mismatch: pre=%v sfs=%v c1=%v", g.GetPreflag(), g.GetScalefacScale(), g.GetCount1TableSelect())
	}
}

func TestReadSideInfo_Stereo_WindowSwitching(t *testing.T) {
	// Stereo, no CRC
	var h MP3FrameHeader
	h.flag1 |= 1 << 6
	h.flag2 = 0

	var bw bitWriter
	// main_data_begin (9) + private_bits (3)
	bw.write(9, 0)
	bw.write(3, 0)
	// SCFSI: two channels
	for i := 0; i < 8; i++ {
		bw.write(1, 0)
	}

	// For each granule/channel: choose window_switching = 1, block_type = short, mixed_block_flag = 0
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < 2; ch++ {
			bw.write(12, 0x010)                 // Part23Length
			bw.write(9, 0x012)                  // BigValues
			bw.write(8, 0x40)                   // GlobalGain
			bw.write(4, 0x5)                    // ScalefacCompress
			bw.write(1, 1)                      // window_switching = 1
			bw.write(2, uint32(BlockTypeShort)) // block_type = short
			bw.write(1, 0)                      // mixed_block_flag = 0
			bw.write(5, 3)                      // TableSelect[0]
			bw.write(5, 4)                      // TableSelect[1]
			// TableSelect[2] not written (forced to 0)
			bw.write(3, 5) // SubblockGain[0]
			bw.write(3, 6) // SubblockGain[1]
			bw.write(3, 7) // SubblockGain[2]
			// Region counts are derived for BlockTypeShort && !mixed_block
			bw.write(1, 0) // preflag
			bw.write(1, 1) // scalefac_scale
			bw.write(1, 0) // count1_table_select
		}
	}

	data := bw.bytes()
	if len(data) != 32 {
		t.Fatalf("built side info length = %d, want 32", len(data))
	}

	r := bufio.NewReader(bytes.NewReader(data))
	si, err := ReadSideInfo(&h, r, GetSideInfoLength(&h))
	if err != nil {
		t.Fatalf("ReadSideInfo failed: %v", err)
	}

	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < 2; ch++ {
			g := si.Granule[gr][ch]
			if !g.GetWindowSwitching() {
				t.Fatalf("granule[%d][%d] expected window switching true", gr, ch)
			}
			if g.GetBlockType() != BlockTypeShort {
				t.Fatalf("granule[%d][%d] block type got %v, want %v", gr, ch, g.GetBlockType(), BlockTypeShort)
			}
			if g.GetMixedBlockFlag() {
				t.Fatalf("granule[%d][%d] mixed block flag got true, want false", gr, ch)
			}
			if g.TableSelect != [3]byte{3, 4, 0} {
				t.Fatalf("granule[%d][%d] TableSelect got %v, want [3 4 0]", gr, ch, g.TableSelect)
			}
			if g.SubblockGain != [3]byte{5, 6, 7} {
				t.Fatalf("granule[%d][%d] SubblockGain got %v, want [5 6 7]", gr, ch, g.SubblockGain)
			}
			if g.Region0Count != 8 || g.Region1Count != 12 {
				t.Fatalf("granule[%d][%d] region counts got (%d,%d), want (8,12)", gr, ch, g.Region0Count, g.Region1Count)
			}
		}
	}
}

func TestReadSideInfo_WindowSwitchingInvalidBlockType(t *testing.T) {
	// Mono, no CRC to minimize size; set invalid combination: window_switching=1 with block_type=long
	var h MP3FrameHeader
	h.flag1 |= 1 << 6 // no CRC
	// ChannelModeMono: top 2 bits of flag2 set to 0b11
	h.flag2 = (byte(ChannelModeMono) << 6)

	var bw bitWriter
	// main_data_begin (9) + private_bits for mono (5)
	bw.write(9, 0)
	bw.write(5, 0)
	// scfsi for mono: 4 bits
	for i := 0; i < 4; i++ {
		bw.write(1, 0)
	}
	// Two granules, one channel
	for gr := 0; gr < 2; gr++ {
		bw.write(12, 0x001)
		bw.write(9, 0x002)
		bw.write(8, 0x10)
		bw.write(4, 0x1)
		bw.write(1, 1)                     // window_switching = 1
		bw.write(2, uint32(BlockTypeLong)) // block_type = long (invalid with window_switching)
		bw.write(1, 0)                     // mixed_block_flag
		// Even though following fields won't be used due to error, pad bits to reach exact length
		bw.write(5, 0)
		bw.write(5, 0)
		bw.write(3, 0)
		// flags
		bw.write(1, 0)
		bw.write(1, 0)
		bw.write(1, 0)
	}

	data := bw.bytes()
	// Ensure buffer length matches expected side info length (mono = 17)
	wantLen := GetSideInfoLength(&h)
	if len(data) < wantLen {
		pad := make([]byte, wantLen-len(data))
		data = append(data, pad...)
	}
	if len(data) != wantLen {
		t.Fatalf("built side info length = %d, want %d", len(data), wantLen)
	}

	r := bufio.NewReader(bytes.NewReader(data))
	_, err := ReadSideInfo(&h, r, GetSideInfoLength(&h))
	if err == nil {
		t.Fatalf("expected error for invalid long block_type with window_switching, got nil")
	}
}
