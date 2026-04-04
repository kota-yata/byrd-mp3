package maindata

import (
	"byrd/internal/common"
	"fmt"
	"math"
)

const (
	mixedLongEndLine  = 36
	mixedShortStartSFB = 3
)

// TODO: float64 is used here for correctness-first implementation; optimize to float32 or fixed-point later if needed.
func Requantize(sampleRate uint16, gc *common.GranuleChannelInfo, scalefactors *Scalefactors, spectralValues []int, out *[]float64) error {
	if gc == nil {
		return fmt.Errorf("nil granule channel info")
	}
	if scalefactors == nil {
		return fmt.Errorf("nil scalefactors")
	}
	if out == nil {
		return fmt.Errorf("nil output buffer")
	}
	if len(spectralValues) != 576 {
		return fmt.Errorf("invalid spectral values length: %d", len(spectralValues))
	}
	sfBands, ok := common.SCALEFACTOR_BAND_INDICES[sampleRate]
	if !ok {
		return fmt.Errorf("unsupported sample rate for scalefactor bands: %d", sampleRate)
	}

	if cap(*out) < 576 {
		*out = make([]float64, 576)
	} else {
		*out = (*out)[:576]
	}
	clear(*out)

	for i, is := range spectralValues {
		if is == 0 {
			continue
		}

		switch {
		case !gc.GetWindowSwitching() || gc.GetBlockType() != common.BlockTypeShort:
			sfb := longSFBForLine(sfBands.Long, i)
			(*out)[i] = requantizeLongLine(is, gc, scalefactors, sfb)
		case gc.GetMixedBlockFlag() && i < mixedLongEndLine:
			sfb := longSFBForLine(sfBands.Long, i)
			(*out)[i] = requantizeLongLine(is, gc, scalefactors, sfb)
		default:
			sfb, win, err := shortSFBWindowForLine(sfBands.Short, i, gc.GetMixedBlockFlag())
			if err != nil {
				return err
			}
			(*out)[i] = requantizeShortLine(is, gc, scalefactors, sfb, win)
		}
	}

	return nil
}

func requantizeLongLine(is int, gc *common.GranuleChannelInfo, scalefactors *Scalefactors, sfb int) float64 {
	pretab := 0
	if gc.GetPreflag() && sfb < len(common.PRETAB) {
		pretab = int(common.PRETAB[sfb])
	}
	scalefacMultiplier := 1
	if gc.GetScalefacScale() {
		scalefacMultiplier = 2
	}

	// xr[i] = sign(is[i]) * |is[i]|^(4/3) *
	//   2^(-(210 - global_gain + 2*(1+scalefac_scale)*(scalefac_l[sfb] + preflag*pretab[sfb])) / 4)
	q := 210 - int(gc.GlobalGain) + 2*scalefacMultiplier*(int(scalefactors.Long[sfb])+pretab)
	return signedPow43(is) * math.Pow(2.0, -float64(q)/4.0)
}

func requantizeShortLine(is int, gc *common.GranuleChannelInfo, scalefactors *Scalefactors, sfb int, win int) float64 {
	scalefacMultiplier := 1
	if gc.GetScalefacScale() {
		scalefacMultiplier = 2
	}

	// xr[i] = sign(is[i]) * |is[i]|^(4/3) *
	//   2^(-(210 - global_gain + 8*subblock_gain[w] + 2*(1+scalefac_scale)*scalefac_s[sfb][w]) / 4)
	q := 210 - int(gc.GlobalGain) + 8*int(gc.SubblockGain[win]) + 2*scalefacMultiplier*int(scalefactors.Short[sfb][win])
	return signedPow43(is) * math.Pow(2.0, -float64(q)/4.0)
}

func signedPow43(is int) float64 {
	mag := math.Pow(math.Abs(float64(is)), 4.0/3.0)
	return math.Copysign(mag, float64(is))
}

func longSFBForLine(longBands [23]int, line int) int {
	for sfb := 0; sfb < len(longBands)-1; sfb++ {
		if line < longBands[sfb+1] {
			return sfb
		}
	}
	return len(longBands) - 2
}

func shortSFBWindowForLine(shortBands [14]int, line int, mixed bool) (int, int, error) {
	shortLine := line
	startSFB := 0
	if mixed {
		if line < mixedLongEndLine {
			return 0, 0, fmt.Errorf("line %d is in mixed long-block region", line)
		}
		shortLine = line - mixedLongEndLine + shortBands[mixedShortStartSFB]*SCALEFACTOR_SHORT_WINDOW_COUNT
		startSFB = mixedShortStartSFB
	}

	for sfb := startSFB; sfb < len(shortBands)-1; sfb++ {
		bandStart := shortBands[sfb] * SCALEFACTOR_SHORT_WINDOW_COUNT
		bandEnd := shortBands[sfb+1] * SCALEFACTOR_SHORT_WINDOW_COUNT
		if shortLine >= bandStart && shortLine < bandEnd {
			width := shortBands[sfb+1] - shortBands[sfb]
			if width <= 0 {
				return 0, 0, fmt.Errorf("invalid short scalefactor band width at sfb %d", sfb)
			}
			window := (shortLine - bandStart) / width
			if window < 0 || window >= SCALEFACTOR_SHORT_WINDOW_COUNT {
				return 0, 0, fmt.Errorf("invalid short window index for line %d", line)
			}
			return sfb, window, nil
		}
	}

	return 0, 0, fmt.Errorf("line %d does not map to a short scalefactor band", line)
}
