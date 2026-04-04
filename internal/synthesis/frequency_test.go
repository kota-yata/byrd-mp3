package synthesis

import "testing"

func TestApplyFrequencyInversion(t *testing.T) {
	var samples [32][18]float64
	samples[0][1] = 1
	samples[1][0] = 2
	samples[1][1] = 3
	samples[1][2] = 4
	samples[2][1] = 5
	samples[3][3] = 6

	ApplyFrequencyInversion(&samples)

	if samples[0][1] != 1 {
		t.Fatalf("even subband should be unchanged, got %f", samples[0][1])
	}
	if samples[1][0] != 2 {
		t.Fatalf("even sample should be unchanged, got %f", samples[1][0])
	}
	if samples[1][1] != -3 {
		t.Fatalf("odd subband odd sample should be inverted, got %f", samples[1][1])
	}
	if samples[1][2] != 4 {
		t.Fatalf("odd subband even sample should be unchanged, got %f", samples[1][2])
	}
	if samples[2][1] != 5 {
		t.Fatalf("even subband should be unchanged, got %f", samples[2][1])
	}
	if samples[3][3] != -6 {
		t.Fatalf("odd subband odd sample should be inverted, got %f", samples[3][3])
	}
}
