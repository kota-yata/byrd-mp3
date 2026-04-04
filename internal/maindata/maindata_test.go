package maindata

import (
	"byrd/internal/common"
	"bytes"
	"testing"
)

type GranuleChannelInfo = common.GranuleChannelInfo
type BlockType = common.BlockType

const BlockTypeShort = common.BlockTypeShort
const PURE_SHORT_REGION0_COUNT = common.PURE_SHORT_REGION0_COUNT
const PURE_SHORT_REGION1_COUNT = common.PURE_SHORT_REGION1_COUNT

var (
	NewBitReader             = common.NewBitReader
	baseTables               = common.BaseTables
	SCALEFACTOR_BAND_INDICES = common.SCALEFACTOR_BAND_INDICES
)

func TestReadMainData_NoReservoir(t *testing.T) {
	// No reservoir, mainDataBegin=0, read N bytes
	cur := []byte("abcdefghij")
	var reservoir []byte
	var mainBuf []byte
	err := ReadMainData(0, &reservoir, cur, &mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}

	if !bytes.Equal(mainBuf, cur) {
		t.Fatalf("main data mismatch: got %q, want %q", string(mainBuf), string(cur))
	}
	if !bytes.Equal(reservoir, cur) {
		t.Fatalf("reservoir mismatch: got %q, want %q", string(reservoir), string(cur))
	}
}

func TestReadMainData_WithReservoirAndBegin(t *testing.T) {
	// Reservoir has 3 bytes; mainDataBegin=2 should pull last 2 bytes
	reservoir := []byte("XYZ")
	cur := []byte("abcde")
	var mainBuf []byte

	err := ReadMainData(2, &reservoir, cur, &mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}

	wantMain := []byte("YZabcde")
	if !bytes.Equal(mainBuf, wantMain) {
		t.Fatalf("main data mismatch: got %q, want %q", string(mainBuf), string(wantMain))
	}
	wantRes := []byte("XYZabcde")
	if !bytes.Equal(reservoir, wantRes) {
		t.Fatalf("reservoir mismatch: got %q, want %q", string(reservoir), string(wantRes))
	}
}

func TestReadMainData_ReservoirUnderflow(t *testing.T) {
	reservoir := []byte("XYZ")
	cur := []byte("abc")
	var mainBuf []byte

	err := ReadMainData(5, &reservoir, cur, &mainBuf) // need 5 bytes, have 3
	if err == nil {
		t.Fatalf("expected reservoir underflow error, got nil")
	}
}

func TestReadMainData_ReservoirTruncation(t *testing.T) {
	// Start with near-limit reservoir, then append more to exceed RESERVOIR_MAX
	if RESERVOIR_MAX < 20 {
		t.Skip("RESERVOIR_MAX too small for truncation test")
	}
	reservoir := bytes.Repeat([]byte{'R'}, RESERVOIR_MAX-5)
	cur := bytes.Repeat([]byte{'C'}, 20)
	var mainBuf []byte

	err := ReadMainData(0, &reservoir, cur, &mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}

	// main should equal current data since mainDataBegin=0
	if !bytes.Equal(mainBuf, cur) {
		t.Fatalf("main data mismatch: got len=%d, want len=%d", len(mainBuf), len(cur))
	}
	if len(reservoir) != RESERVOIR_MAX {
		t.Fatalf("reservoir length got %d, want %d", len(reservoir), RESERVOIR_MAX)
	}
	// Last RESERVOIR_MAX bytes should end with the newly appended 'C's
	for i := 0; i < len(cur); i++ {
		if reservoir[len(reservoir)-len(cur)+i] != 'C' {
			t.Fatalf("reservoir tail mismatch at %d", i)
		}
	}
}

func TestReadMainData_ReusesMainDataBuffer(t *testing.T) {
	reservoir := []byte("XYZ")
	cur := []byte("abcde")
	mainBuf := make([]byte, 0, 16)

	err := ReadMainData(2, &reservoir, cur, &mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}
	if len(mainBuf) == 0 {
		t.Fatalf("main data is empty")
	}
	if &mainBuf[0] != &mainBuf[:cap(mainBuf)][0] {
		t.Fatalf("expected ReadMainData to reuse provided buffer")
	}
}

func TestParseScaleFactor_LongBlockGranule0(t *testing.T) {
	var bw bitWriter
	wantLong := [21]uint8{}
	for i := 0; i <= 10; i++ {
		v := uint8(i % 2)
		wantLong[i] = v
		bw.write(1, uint32(v))
	}
	for i := 11; i <= 20; i++ {
		v := uint8((i - 11) % 8)
		wantLong[i] = v
		bw.write(3, uint32(v))
	}

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{Part23Length: 41, ScalefacCompress: 7}
	var got Scalefactors

	bits, err := ParseScaleFactor(br, gc, [4]byte{}, 0, nil, &got)
	if err != nil {
		t.Fatalf("ParseScaleFactor failed: %v", err)
	}
	if bits != 41 {
		t.Fatalf("bits consumed = %d, want 41", bits)
	}
	if got.Long != wantLong {
		t.Fatalf("long scalefactors got %v, want %v", got.Long, wantLong)
	}
}

func TestParseScaleFactor_LongBlockGranule1SCFSIReuse(t *testing.T) {
	prev := &Scalefactors{}
	for i := range prev.Long {
		prev.Long[i] = uint8(20 + i)
	}

	var bw bitWriter
	wantLong := prev.Long
	for i := 6; i <= 10; i++ {
		v := uint8((i - 6) % 4)
		wantLong[i] = v
		bw.write(2, uint32(v))
	}
	for i := 16; i <= 20; i++ {
		v := uint8((i - 16) % 2)
		wantLong[i] = v
		bw.write(1, uint32(v))
	}

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{Part23Length: 15, ScalefacCompress: 8}
	var got Scalefactors

	bits, err := ParseScaleFactor(br, gc, [4]byte{1, 0, 1, 0}, 1, prev, &got)
	if err != nil {
		t.Fatalf("ParseScaleFactor failed: %v", err)
	}
	if bits != 15 {
		t.Fatalf("bits consumed = %d, want 15", bits)
	}
	if got.Long != wantLong {
		t.Fatalf("long scalefactors got %v, want %v", got.Long, wantLong)
	}
}

func TestParseScaleFactor_ShortBlock(t *testing.T) {
	var bw bitWriter
	var wantShort [12][3]uint8
	for sfb := 0; sfb <= 5; sfb++ {
		for win := 0; win < 3; win++ {
			v := uint8((sfb + win) % 2)
			wantShort[sfb][win] = v
			bw.write(1, uint32(v))
		}
	}
	for sfb := 6; sfb <= 11; sfb++ {
		for win := 0; win < 3; win++ {
			v := uint8((sfb + win) % 4)
			wantShort[sfb][win] = v
			bw.write(2, uint32(v))
		}
	}

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{Part23Length: 54, ScalefacCompress: 6}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(BlockTypeShort)
	var got Scalefactors

	bits, err := ParseScaleFactor(br, gc, [4]byte{}, 0, nil, &got)
	if err != nil {
		t.Fatalf("ParseScaleFactor failed: %v", err)
	}
	if bits != 54 {
		t.Fatalf("bits consumed = %d, want 54", bits)
	}
	if got.Short != wantShort {
		t.Fatalf("short scalefactors got %v, want %v", got.Short, wantShort)
	}
}

func TestParseScaleFactor_MixedBlock(t *testing.T) {
	var bw bitWriter
	var want Scalefactors
	for sfb := 0; sfb <= 7; sfb++ {
		v := uint8(sfb)
		want.Long[sfb] = v
		bw.write(4, uint32(v))
	}
	for sfb := 3; sfb <= 5; sfb++ {
		for win := 0; win < 3; win++ {
			v := uint8((sfb + win) & 0xF)
			want.Short[sfb][win] = v
			bw.write(4, uint32(v))
		}
	}
	for sfb := 6; sfb <= 11; sfb++ {
		for win := 0; win < 3; win++ {
			v := uint8((sfb + win) % 4)
			want.Short[sfb][win] = v
			bw.write(2, uint32(v))
		}
	}

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{Part23Length: 104, ScalefacCompress: 14}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(BlockTypeShort)
	gc.SetMixedBlockFlag(true)
	var got Scalefactors

	bits, err := ParseScaleFactor(br, gc, [4]byte{}, 0, nil, &got)
	if err != nil {
		t.Fatalf("ParseScaleFactor failed: %v", err)
	}
	if bits != 104 {
		t.Fatalf("bits consumed = %d, want 104", bits)
	}
	if got != want {
		t.Fatalf("mixed scalefactors got %+v, want %+v", got, want)
	}
}

func TestParseScaleFactor_Part23TooShort(t *testing.T) {
	var bw bitWriter
	for i := 0; i <= 10; i++ {
		bw.write(1, 1)
	}
	for i := 11; i <= 20; i++ {
		bw.write(3, 1)
	}

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{Part23Length: 40, ScalefacCompress: 7}
	var got Scalefactors

	_, err := ParseScaleFactor(br, gc, [4]byte{}, 0, nil, &got)
	if err == nil {
		t.Fatalf("expected part23 length error, got nil")
	}
}
