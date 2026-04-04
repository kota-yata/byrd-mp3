package hybrid

import (
	"byrd/internal/common"
	"testing"
)

func TestHybridSynthesis_ZeroInput(t *testing.T) {
	gc := &common.GranuleChannelInfo{}
	values := make([]float64, 576)
	var overlap [32][18]float64
	var out [32][18]float64

	if err := HybridSynthesis(gc, values, &overlap, &out); err != nil {
		t.Fatalf("HybridSynthesis failed: %v", err)
	}

	for sb := range out {
		for i := range out[sb] {
			if out[sb][i] != 0 || overlap[sb][i] != 0 {
				t.Fatalf("zero input should keep zero state, got out=%f overlap=%f", out[sb][i], overlap[sb][i])
			}
		}
	}
}

func TestHybridSynthesis_LongBlockOverlap(t *testing.T) {
	gc := &common.GranuleChannelInfo{}
	values := make([]float64, 576)
	values[0] = 1
	var overlap [32][18]float64
	var out [32][18]float64

	if err := HybridSynthesis(gc, values, &overlap, &out); err != nil {
		t.Fatalf("HybridSynthesis failed: %v", err)
	}

	nonZeroOut := 0
	nonZeroOverlap := 0
	for i := 0; i < 18; i++ {
		if out[0][i] != 0 {
			nonZeroOut++
		}
		if overlap[0][i] != 0 {
			nonZeroOverlap++
		}
	}
	if nonZeroOut == 0 || nonZeroOverlap == 0 {
		t.Fatalf("expected non-zero output and overlap, got out=%d overlap=%d", nonZeroOut, nonZeroOverlap)
	}

	values = make([]float64, 576)
	var next [32][18]float64
	if err := HybridSynthesis(gc, values, &overlap, &next); err != nil {
		t.Fatalf("HybridSynthesis second call failed: %v", err)
	}
	nonZeroNext := 0
	for i := 0; i < 18; i++ {
		if next[0][i] != 0 {
			nonZeroNext++
		}
	}
	if nonZeroNext == 0 {
		t.Fatalf("expected overlap-add output on second call")
	}
}

func TestHybridSynthesis_ShortBlock(t *testing.T) {
	gc := &common.GranuleChannelInfo{}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(common.BlockTypeShort)
	values := make([]float64, 576)
	values[0] = 1
	values[1] = 2
	values[2] = 3
	var overlap [32][18]float64
	var out [32][18]float64

	if err := HybridSynthesis(gc, values, &overlap, &out); err != nil {
		t.Fatalf("HybridSynthesis failed: %v", err)
	}

	nonZero := 0
	for i := 0; i < 18; i++ {
		if out[0][i] != 0 || overlap[0][i] != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatalf("expected non-zero short block synthesis")
	}
}

func TestHybridSynthesis_MixedBlockUsesLongForFirstSubbands(t *testing.T) {
	gc := &common.GranuleChannelInfo{}
	gc.SetWindowSwitching(true)
	gc.SetBlockType(common.BlockTypeShort)
	gc.SetMixedBlockFlag(true)
	values := make([]float64, 576)
	values[17] = 1
	values[35] = 1
	values[36] = 1
	var overlap [32][18]float64
	var out [32][18]float64

	if err := HybridSynthesis(gc, values, &overlap, &out); err != nil {
		t.Fatalf("HybridSynthesis failed: %v", err)
	}

	if out[0][0] == 0 && out[1][0] == 0 && out[2][0] == 0 {
		t.Fatalf("expected mixed block synthesis to produce output")
	}
}
