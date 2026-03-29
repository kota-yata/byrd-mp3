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
	main, err := ReadMainData(0, &reservoir, cur, mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}

	if !bytes.Equal(main, cur) {
		t.Fatalf("main data mismatch: got %q, want %q", string(main), string(cur))
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

	main, err := ReadMainData(2, &reservoir, cur, mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}

	wantMain := []byte("YZabcde")
	if !bytes.Equal(main, wantMain) {
		t.Fatalf("main data mismatch: got %q, want %q", string(main), string(wantMain))
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

	_, err := ReadMainData(5, &reservoir, cur, mainBuf) // need 5 bytes, have 3
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

	main, err := ReadMainData(0, &reservoir, cur, mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}

	// main should equal current data since mainDataBegin=0
	if !bytes.Equal(main, cur) {
		t.Fatalf("main data mismatch: got len=%d, want len=%d", len(main), len(cur))
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

	main, err := ReadMainData(2, &reservoir, cur, mainBuf)
	if err != nil {
		t.Fatalf("ReadMainData failed: %v", err)
	}
	if len(main) == 0 {
		t.Fatalf("main data is empty")
	}
	if &main[0] != &mainBuf[:cap(mainBuf)][0] {
		t.Fatalf("expected ReadMainData to reuse provided buffer")
	}
}
