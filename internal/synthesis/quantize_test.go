package synthesis

import "testing"

func TestQuantizeSample(t *testing.T) {
	if got := QuantizeSample(0); got != 0 {
		t.Fatalf("zero got %d, want 0", got)
	}
	if got := QuantizeSample(1); got != 32767 {
		t.Fatalf("one got %d, want 32767", got)
	}
	if got := QuantizeSample(-1); got != -32767 {
		t.Fatalf("minus one got %d, want -32767", got)
	}
	if got := QuantizeSample(2); got != 32767 {
		t.Fatalf("positive clip got %d, want 32767", got)
	}
	if got := QuantizeSample(-2); got != -32768 {
		t.Fatalf("negative clip got %d, want -32768", got)
	}
}
