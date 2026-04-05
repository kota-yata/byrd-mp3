package maindata

import (
	"github.com/kota-yata/byrd-mp3/internal/common"
	"testing"
)

func sameTable(got *common.HuffmanTable, want common.HuffmanTable) bool {
	if got == nil {
		return false
	}
	if got.Linbits != want.Linbits || got.TreeLen != want.TreeLen {
		return false
	}
	if len(got.Data) == 0 || len(want.Data) == 0 {
		return len(got.Data) == len(want.Data)
	}
	return &got.Data[0] == &want.Data[0]
}

func nextHuffmanIndex(table *common.HuffmanTable, idx int, bit uint32) int {
	if bit != 0 {
		for (table.Data[idx] & 0x00FF) >= 250 {
			idx += int(table.Data[idx] & 0x00FF)
		}
		return idx + int(table.Data[idx]&0x00FF)
	}
	for (table.Data[idx] >> 8) >= 250 {
		idx += int(table.Data[idx] >> 8)
	}
	return idx + int(table.Data[idx]>>8)
}

func findPairCode(t *testing.T, table common.HuffmanTable, wantX int, wantY int) []uint32 {
	t.Helper()

	var search func(idx int, path []uint32) ([]uint32, bool)
	search = func(idx int, path []uint32) ([]uint32, bool) {
		if idx < 0 || idx >= len(table.Data) {
			return nil, false
		}
		node := table.Data[idx]
		if isHuffmanLeaf(node) {
			x := int((node >> 4) & 0xF)
			y := int(node & 0xF)
			if x == wantX && y == wantY {
				out := make([]uint32, len(path))
				copy(out, path)
				return out, true
			}
			return nil, false
		}
		for _, bit := range []uint32{0, 1} {
			next := nextHuffmanIndex(&table, idx, bit)
			if code, ok := search(next, append(path, bit)); ok {
				return code, true
			}
		}
		return nil, false
	}

	code, ok := search(0, nil)
	if !ok {
		t.Fatalf("pair (%d,%d) not found in table", wantX, wantY)
	}
	return code
}

func findQuadCode(t *testing.T, table common.HuffmanTable, want [4]int) []uint32 {
	t.Helper()

	var search func(idx int, path []uint32) ([]uint32, bool)
	search = func(idx int, path []uint32) ([]uint32, bool) {
		if idx < 0 || idx >= len(table.Data) {
			return nil, false
		}
		node := table.Data[idx]
		if isHuffmanLeaf(node) {
			got := [4]int{
				int((node >> 3) & 0x1),
				int((node >> 2) & 0x1),
				int((node >> 1) & 0x1),
				int(node & 0x1),
			}
			if got == want {
				out := make([]uint32, len(path))
				copy(out, path)
				return out, true
			}
			return nil, false
		}
		for _, bit := range []uint32{0, 1} {
			next := nextHuffmanIndex(&table, idx, bit)
			if code, ok := search(next, append(path, bit)); ok {
				return code, true
			}
		}
		return nil, false
	}

	code, ok := search(0, nil)
	if !ok {
		t.Fatalf("quad %v not found in table", want)
	}
	return code
}

func TestSelectTable_LongBlock(t *testing.T) {
	gc := &common.GranuleChannelInfo{
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
		{29, 7},
		{30, 9},
	} {
		got, err := selectTable(44100, gc, tc.lineIndex)
		if err != nil {
			t.Fatalf("selectTable(%d) failed: %v", tc.lineIndex, err)
		}
		want := common.BaseTables[tc.wantTable]
		if !sameTable(got, want) {
			t.Fatalf("selectTable(%d) got table %+v, want table %d", tc.lineIndex, *got, tc.wantTable)
		}
	}
}

func TestSelectTable_LongBlock_Region1CountZero(t *testing.T) {
	gc := &common.GranuleChannelInfo{
		TableSelect:  [3]byte{5, 7, 9},
		Region0Count: 1,
		Region1Count: 0,
	}

	got, err := selectTable(48000, gc, 8)
	if err != nil {
		t.Fatalf("selectTable failed: %v", err)
	}
	want := common.BaseTables[7]
	if !sameTable(got, want) {
		t.Fatalf("line 8 got table %+v, want table 7", *got)
	}
}

func TestSelectTable_SwitchedWindow(t *testing.T) {
	gc := &common.GranuleChannelInfo{
		TableSelect:  [3]byte{16, 24, 0},
		Region0Count: common.PURE_SHORT_REGION0_COUNT,
		Region1Count: common.PURE_SHORT_REGION1_COUNT,
	}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(common.BlockTypeShort)

	got0, err := selectTable(44100, gc, 0)
	if err != nil {
		t.Fatalf("selectTable region0 failed: %v", err)
	}
	if got0.Linbits != 1 || len(got0.Data) != len(common.BaseTables[16].Data) {
		t.Fatalf("region0 got %+v, want table 16", *got0)
	}
	if !sameTable(got0, common.BaseTables[16]) {
		t.Fatalf("region0 got %+v, want table 16", *got0)
	}

	got1, err := selectTable(44100, gc, 36)
	if err != nil {
		t.Fatalf("selectTable region1 failed: %v", err)
	}
	if got1.Linbits != 4 || len(got1.Data) != len(common.BaseTables[24].Data) {
		t.Fatalf("region1 got %+v, want table 24", *got1)
	}
	if !sameTable(got1, common.BaseTables[24]) {
		t.Fatalf("region1 got %+v, want table 24", *got1)
	}
}

func TestSelectTable_StartBlock_UsesLongRegions(t *testing.T) {
	gc := &common.GranuleChannelInfo{
		TableSelect:  [3]byte{5, 7, 9},
		Region0Count: 2,
		Region1Count: 3,
	}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(common.BlockTypeStart)

	for _, tc := range []struct {
		lineIndex int
		wantTable int
	}{
		{0, 5},
		{11, 5},
		{12, 7},
		{29, 7},
		{30, 9},
	} {
		got, err := selectTable(44100, gc, tc.lineIndex)
		if err != nil {
			t.Fatalf("selectTable(%d) failed: %v", tc.lineIndex, err)
		}
		want := common.BaseTables[tc.wantTable]
		if !sameTable(got, want) {
			t.Fatalf("selectTable(%d) got table %+v, want table %d", tc.lineIndex, *got, tc.wantTable)
		}
	}
}

func TestSelectTable_Invalid(t *testing.T) {
	if _, err := selectTable(44100, nil, 0); err == nil {
		t.Fatalf("expected nil granule channel error")
	}

	gc := &common.GranuleChannelInfo{TableSelect: [3]byte{4, 7, 9}}
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

func TestGuardedReadBit_RespectsLimit(t *testing.T) {
	br := common.NewBitReader([]byte{0x80})
	var scratch uint32

	if err := guardedReadBit(br, 1, &scratch); err != nil {
		t.Fatalf("guardedReadBit failed: %v", err)
	}
	if scratch != 1 {
		t.Fatalf("scratch got %d, want 1", scratch)
	}
	if err := guardedReadBit(br, 1, &scratch); err == nil {
		t.Fatalf("expected limit error")
	}
}

func TestGuardedReadBits_RespectsLimit(t *testing.T) {
	br := common.NewBitReader([]byte{0xE0})
	var scratch uint32

	if err := guardedReadBits(br, 3, 3, &scratch); err != nil {
		t.Fatalf("guardedReadBits failed: %v", err)
	}
	if scratch != 0b111 {
		t.Fatalf("scratch got %b, want 111", scratch)
	}
	if err := guardedReadBits(br, 3, 1, &scratch); err == nil {
		t.Fatalf("expected limit error")
	}
}

func TestDecodeHuffmanPair_Table1(t *testing.T) {
	table := common.BaseTables[1]
	code := findPairCode(t, table, 1, 0)
	var bw bitWriter
	for _, bit := range code {
		bw.write(1, bit)
	}
	bw.write(1, 0) // x sign bit

	br := common.NewBitReader(bw.bytes())
	var scratch uint32
	x, y, err := decodeHuffmanPair(br, &table, len(code)+1, &scratch)
	if err != nil {
		t.Fatalf("decodeHuffmanPair failed: %v", err)
	}
	if x != 1 || y != 0 {
		t.Fatalf("decoded pair got (%d,%d), want (1,0)", x, y)
	}
}

func TestDecodeHuffmanPair_Linbits(t *testing.T) {
	table := common.BaseTables[16]
	code := findPairCode(t, table, 15, 15)
	var bw bitWriter
	for _, bit := range code {
		bw.write(1, bit)
	}
	bw.write(1, 0b1) // x linbit
	bw.write(1, 0b0) // x sign bit
	bw.write(1, 0b0) // y linbit
	bw.write(1, 0b0) // y sign bit

	br := common.NewBitReader(bw.bytes())
	var scratch uint32
	x, y, err := decodeHuffmanPair(br, &table, len(code)+4, &scratch)
	if err != nil {
		t.Fatalf("decodeHuffmanPair failed: %v", err)
	}
	if x != 16 || y != 15 {
		t.Fatalf("decoded pair got (%d,%d), want (16,15)", x, y)
	}
}

func TestDecodeHuffmanPair_Invalid(t *testing.T) {
	br := common.NewBitReader([]byte{0x00})
	var scratch uint32
	if _, _, err := decodeHuffmanPair(br, nil, 1, &scratch); err == nil {
		t.Fatalf("expected nil table error")
	}
	empty := common.HuffmanTable{}
	if _, _, err := decodeHuffmanPair(br, &empty, 1, &scratch); err == nil {
		t.Fatalf("expected empty table error")
	}
}

func TestDecodeHuffmanQuad_Table33(t *testing.T) {
	table := common.BaseTables[33]
	code := findQuadCode(t, table, [4]int{0, 1, 0, 1})
	var bw bitWriter
	for _, bit := range code {
		bw.write(1, bit)
	}
	bw.write(1, 0) // w sign bit
	bw.write(1, 0) // y sign bit

	br := common.NewBitReader(bw.bytes())
	var scratch uint32
	v, w, x, y, err := decodeHuffmanQuad(br, &table, len(code)+2, &scratch)
	if err != nil {
		t.Fatalf("decodeHuffmanQuad failed: %v", err)
	}
	if [4]int{v, w, x, y} != [4]int{0, 1, 0, 1} {
		t.Fatalf("decoded quad got %v, want [0 1 0 1]", [4]int{v, w, x, y})
	}
}

func TestDecodeHuffmanQuad_Invalid(t *testing.T) {
	br := common.NewBitReader([]byte{0x00})
	var scratch uint32
	if _, _, _, _, err := decodeHuffmanQuad(br, nil, 1, &scratch); err == nil {
		t.Fatalf("expected nil table error")
	}
	empty := common.HuffmanTable{}
	if _, _, _, _, err := decodeHuffmanQuad(br, &empty, 1, &scratch); err == nil {
		t.Fatalf("expected empty table error")
	}
}

func TestParseBigValues_Table1(t *testing.T) {
	table := common.BaseTables[1]
	var bw bitWriter
	for _, bit := range findPairCode(t, table, 1, 0) {
		bw.write(1, bit)
	}
	bw.write(1, 1)
	for _, bit := range findPairCode(t, table, 0, 1) {
		bw.write(1, bit)
	}
	bw.write(1, 0)
	for _, bit := range findPairCode(t, table, 1, 1) {
		bw.write(1, bit)
	}
	bw.write(1, 1)
	bw.write(1, 0)

	br := common.NewBitReader(bw.bytes())
	gc := &common.GranuleChannelInfo{
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
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("decoded big values prefix got %v, want %v", got[:len(want)], want)
		}
	}
}

func TestParseBigValues_RespectsPart23End(t *testing.T) {
	table := common.BaseTables[1]
	var bw bitWriter
	for _, bit := range findPairCode(t, table, 1, 0) {
		bw.write(1, bit)
	}
	bw.write(1, 1)

	br := common.NewBitReader(bw.bytes())
	gc := &common.GranuleChannelInfo{
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
	table := common.BaseTables[33]
	var bw bitWriter
	for _, bit := range findQuadCode(t, table, [4]int{0, 1, 0, 1}) {
		bw.write(1, bit)
	}
	bw.write(1, 1)
	bw.write(1, 0)

	br := common.NewBitReader(bw.bytes())
	gc := &common.GranuleChannelInfo{}
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
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("decoded count1 values prefix got %v, want %v", got[:len(want)], want)
		}
	}
}

func TestParseCount1Values_RespectsPart23End(t *testing.T) {
	table := common.BaseTables[33]
	var bw bitWriter
	for _, bit := range findQuadCode(t, table, [4]int{1, 1, 1, 1}) {
		bw.write(1, bit)
	}

	br := common.NewBitReader(bw.bytes())
	gc := &common.GranuleChannelInfo{}
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
