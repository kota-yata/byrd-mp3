package synthesis

import "testing"

func TestSynthesizeSubbandSamples_ZeroInput(t *testing.T) {
	var state PolyphaseState
	in := make([]float32, 32)
	out := make([]float32, 32)

	if err := SynthesizeSubbandSamples(in, &state, out); err != nil {
		t.Fatalf("SynthesizeSubbandSamples failed: %v", err)
	}
	for _, v := range out {
		if v != 0 {
			t.Fatalf("zero input should produce zero output, got %f", v)
		}
	}
}

func TestSynthesizeSubbandSamples_Stateful(t *testing.T) {
	var state PolyphaseState
	in := make([]float32, 32)
	in[0] = 1
	out := make([]float32, 32)

	if err := SynthesizeSubbandSamples(in, &state, out); err != nil {
		t.Fatalf("SynthesizeSubbandSamples failed: %v", err)
	}
	nonZero := 0
	for _, v := range out {
		if v != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatalf("expected non-zero output")
	}

	zeroIn := make([]float32, 32)
	next := make([]float32, 32)
	if err := SynthesizeSubbandSamples(zeroIn, &state, next); err != nil {
		t.Fatalf("SynthesizeSubbandSamples second call failed: %v", err)
	}
	nonZeroNext := 0
	for _, v := range next {
		if v != 0 {
			nonZeroNext++
		}
	}
	if nonZeroNext == 0 {
		t.Fatalf("expected stateful non-zero output on second call")
	}
}

func TestSynthesizeGranule(t *testing.T) {
	var in [32][18]float32
	in[0][0] = 1
	in[1][1] = 1
	var state PolyphaseState
	var out [576]float32

	if err := SynthesizeGranule(&in, &state, &out); err != nil {
		t.Fatalf("SynthesizeGranule failed: %v", err)
	}

	nonZero := 0
	for _, v := range out {
		if v != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatalf("expected non-zero granule output")
	}
}
