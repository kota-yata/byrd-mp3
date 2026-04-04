package byrd

import (
	"bytes"
	"testing"
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

func TestSelectTable_LongBlock(t *testing.T) {
	gc := &GranuleChannelInfo{
		TableSelect:  [3]byte{5, 7, 9},
		Region0Count: 2,
		Region1Count: 3,
	}

	for _, tc := range []struct {
		lineIndex int
		wantTable int
	}{
		{0, 5},
		{11, 5},
		{12, 7},
		{23, 7},
		{24, 9},
	} {
		got, err := selectTable(44100, gc, tc.lineIndex)
		if err != nil {
			t.Fatalf("selectTable(%d) failed: %v", tc.lineIndex, err)
		}
		if got.Linbits != baseTables[tc.wantTable].Linbits || len(got.Data) != len(baseTables[tc.wantTable].Data) {
			t.Fatalf("selectTable(%d) got table %+v, want table %d", tc.lineIndex, *got, tc.wantTable)
		}
	}
}

func TestSelectTable_LongBlock_Region1CountZero(t *testing.T) {
	gc := &GranuleChannelInfo{
		TableSelect:  [3]byte{5, 7, 9},
		Region0Count: 1,
		Region1Count: 0,
	}

	got, err := selectTable(48000, gc, 8)
	if err != nil {
		t.Fatalf("selectTable failed: %v", err)
	}
	if got.Linbits != baseTables[9].Linbits || len(got.Data) != len(baseTables[9].Data) {
		t.Fatalf("line 8 got table %+v, want table 9", *got)
	}
}

func TestSelectTable_SwitchedWindow(t *testing.T) {
	gc := &GranuleChannelInfo{
		TableSelect:  [3]byte{16, 24, 0},
		Region0Count: PURE_SHORT_REGION0_COUNT,
		Region1Count: PURE_SHORT_REGION1_COUNT,
	}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(BlockTypeShort)

	got0, err := selectTable(44100, gc, 0)
	if err != nil {
		t.Fatalf("selectTable region0 failed: %v", err)
	}
	if got0.Linbits != 1 || len(got0.Data) != len(baseTables[16].Data) {
		t.Fatalf("region0 got %+v, want table 16", *got0)
	}

	got1, err := selectTable(44100, gc, 36)
	if err != nil {
		t.Fatalf("selectTable region1 failed: %v", err)
	}
	if got1.Linbits != 4 || len(got1.Data) != len(baseTables[24].Data) {
		t.Fatalf("region1 got %+v, want table 24", *got1)
	}
}

func TestSelectTable_Invalid(t *testing.T) {
	if _, err := selectTable(44100, nil, 0); err == nil {
		t.Fatalf("expected nil granule channel error")
	}

	gc := &GranuleChannelInfo{TableSelect: [3]byte{4, 7, 9}}
	if _, err := selectTable(44100, gc, -1); err == nil {
		t.Fatalf("expected negative index error")
	}
	if _, err := selectTable(12345, gc, 0); err == nil {
		t.Fatalf("expected unsupported sample rate error")
	}
	if _, err := selectTable(44100, gc, 0); err == nil {
		t.Fatalf("expected unsupported table error")
	}
}

func TestParseBigValues_Table1(t *testing.T) {
	var bw bitWriter
	// table1 codes:
	// 01  + sign(1)          -> (-1, 0)
	// 001 + sign(0)          -> (0, 1)
	// 000 + sign(1) + sign(0)-> (-1, 1)
	bw.write(2, 0b01)
	bw.write(1, 1)
	bw.write(3, 0b001)
	bw.write(1, 0)
	bw.write(3, 0b000)
	bw.write(1, 1)
	bw.write(1, 0)

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{
		BigValues:    3,
		TableSelect:  [3]byte{1, 1, 1},
		Region0Count: 10,
		Region1Count: 10,
	}
	got := make([]int, 576)

	lines, err := ParseBigValues(br, 44100, gc, 12, &got)
	if err != nil {
		t.Fatalf("ParseBigValues failed: %v", err)
	}
	if lines != 6 {
		t.Fatalf("decoded line count = %d, want 6", lines)
	}
	want := []int{-1, 0, 0, 1, -1, 1}
	if len(got) != 576 {
		t.Fatalf("decoded big values length = %d, want 576", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("decoded big values prefix got %v, want %v", got[:len(want)], want)
		}
	}
	for i := len(want); i < len(got); i++ {
		if got[i] != 0 {
			t.Fatalf("decoded big values should remain zero padded at %d: got %d", i, got[i])
		}
	}
}

func TestParseBigValues_RespectsPart23End(t *testing.T) {
	var bw bitWriter
	bw.write(2, 0b01)
	bw.write(1, 1)

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{
		BigValues:    1,
		TableSelect:  [3]byte{1, 1, 1},
		Region0Count: 10,
		Region1Count: 10,
	}
	got := make([]int, 576)

	_, err := ParseBigValues(br, 44100, gc, 2, &got)
	if err == nil {
		t.Fatalf("expected part23 limit error, got nil")
	}
}

func TestParseCount1Values_Table33(t *testing.T) {
	var bw bitWriter
	// table 33 is a balanced 4-bit code; 1010 decodes leaf 0b0101 with this tree.
	bw.write(4, 0b1010)
	bw.write(1, 1) // w sign
	bw.write(1, 0) // y sign

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{}
	gc.SetCount1TableSelect(true)
	got := make([]int, 576)
	got[0] = 9
	got[1] = 8
	gc.BigValues = 1

	lines, err := ParseCount1Values(br, gc, 6, &got)
	if err != nil {
		t.Fatalf("ParseCount1Values failed: %v", err)
	}
	if lines != 4 {
		t.Fatalf("decoded line count = %d, want 4", lines)
	}
	want := []int{9, 8, 0, -1, 0, 1}
	if len(got) != 576 {
		t.Fatalf("decoded count1 values length = %d, want 576", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("decoded count1 values prefix got %v, want %v", got[:len(want)], want)
		}
	}
	for i := len(want); i < len(got); i++ {
		if got[i] != 0 {
			t.Fatalf("decoded count1 values should remain zero padded at %d: got %d", i, got[i])
		}
	}
}

func TestParseCount1Values_RespectsPart23End(t *testing.T) {
	var bw bitWriter
	bw.write(4, 0b1111)

	br := NewBitReader(bw.bytes())
	gc := &GranuleChannelInfo{}
	gc.SetCount1TableSelect(true)
	got := make([]int, 576)

	lines, err := ParseCount1Values(br, gc, 3, &got)
	if err != nil {
		t.Fatalf("ParseCount1Values failed: %v", err)
	}
	if lines != 0 {
		t.Fatalf("decoded line count = %d, want 0", lines)
	}
}
