package byrd

import "testing"

func TestBaseTables(t *testing.T) {
	for i := 1; i <= 33; i++ {
		if _, ok := baseTables[i]; !ok {
			t.Fatalf("baseTables missing key %d", i)
		}
	}

	if got := baseTables[4]; got.Data != nil || got.Linbits != 0 {
		t.Fatalf("table 4 got %+v, want nil data and linbits 0", got)
	}
	if got := baseTables[14]; got.Data != nil || got.Linbits != 0 {
		t.Fatalf("table 14 got %+v, want nil data and linbits 0", got)
	}

	if len(baseTables[16].Data) != len(baseTables[23].Data) {
		t.Fatalf("table 16/23 data length mismatch: %d vs %d", len(baseTables[16].Data), len(baseTables[23].Data))
	}
	if len(baseTables[24].Data) != len(baseTables[31].Data) {
		t.Fatalf("table 24/31 data length mismatch: %d vs %d", len(baseTables[24].Data), len(baseTables[31].Data))
	}

	for _, tc := range []struct {
		table   int
		linbits int
	}{
		{16, 1},
		{17, 2},
		{23, 13},
		{24, 4},
		{31, 13},
		{32, 0},
		{33, 0},
	} {
		if got := baseTables[tc.table].Linbits; got != tc.linbits {
			t.Fatalf("table %d linbits got %d, want %d", tc.table, got, tc.linbits)
		}
	}

	for _, tc := range []struct {
		table int
		first uint16
		size  int
	}{
		{1, 0x0201, 7},
		{2, 0x0201, 17},
		{16, 0x0201, 511},
		{24, 0x3c01, 512},
		{32, 0x0201, 31},
		{33, 0x1001, 31},
	} {
		got := baseTables[tc.table]
		if len(got.Data) != tc.size {
			t.Fatalf("table %d size got %d, want %d", tc.table, len(got.Data), tc.size)
		}
		if len(got.Data) == 0 || got.Data[0] != tc.first {
			t.Fatalf("table %d first value got %#x, want %#x", tc.table, got.Data[0], tc.first)
		}
	}
}
