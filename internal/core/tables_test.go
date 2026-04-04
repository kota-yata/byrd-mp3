package core

import "testing"

func TestBaseTables(t *testing.T) {
	for i := 1; i <= 33; i++ {
		if _, ok := BaseTables[i]; !ok {
			t.Fatalf("baseTables missing key %d", i)
		}
	}

	if got := BaseTables[4]; got.Data != nil || got.Linbits != 0 {
		t.Fatalf("table 4 got %+v, want nil data and linbits 0", got)
	}
	if got := BaseTables[14]; got.Data != nil || got.Linbits != 0 {
		t.Fatalf("table 14 got %+v, want nil data and linbits 0", got)
	}

	if len(BaseTables[16].Data) != len(BaseTables[23].Data) {
		t.Fatalf("table 16/23 data length mismatch: %d vs %d", len(BaseTables[16].Data), len(BaseTables[23].Data))
	}
	if len(BaseTables[24].Data) != len(BaseTables[31].Data) {
		t.Fatalf("table 24/31 data length mismatch: %d vs %d", len(BaseTables[24].Data), len(BaseTables[31].Data))
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
		if got := BaseTables[tc.table].Linbits; got != tc.linbits {
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
		got := BaseTables[tc.table]
		if len(got.Data) != tc.size {
			t.Fatalf("table %d size got %d, want %d", tc.table, len(got.Data), tc.size)
		}
		if len(got.Data) == 0 || got.Data[0] != tc.first {
			t.Fatalf("table %d first value got %#x, want %#x", tc.table, got.Data[0], tc.first)
		}
	}
}

func TestScalefactorBandIndices(t *testing.T) {
	for _, tc := range []struct {
		sampleRate uint16
		long8      int
		long21     int
		short3     int
		short12    int
	}{
		{32000, 36, 550, 12, 136},
		{44100, 36, 418, 12, 136},
		{48000, 36, 384, 12, 126},
	} {
		bands, ok := SCALEFACTOR_BAND_INDICES[tc.sampleRate]
		if !ok {
			t.Fatalf("missing scalefactor band indices for %d", tc.sampleRate)
		}
		if bands.Long[8] != tc.long8 || bands.Long[21] != tc.long21 {
			t.Fatalf("sampleRate %d long bands got [%d %d], want [%d %d]", tc.sampleRate, bands.Long[8], bands.Long[21], tc.long8, tc.long21)
		}
		if bands.Short[3] != tc.short3 || bands.Short[12] != tc.short12 {
			t.Fatalf("sampleRate %d short bands got [%d %d], want [%d %d]", tc.sampleRate, bands.Short[3], bands.Short[12], tc.short3, tc.short12)
		}
		if bands.Long[22] != 576 || bands.Short[13] != 192 {
			t.Fatalf("sampleRate %d terminal bands got long=%d short=%d, want long=576 short=192", tc.sampleRate, bands.Long[22], bands.Short[13])
		}
	}
}
