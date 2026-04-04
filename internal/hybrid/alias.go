package hybrid

import (
	"byrd/internal/common"
	"fmt"
)

// cosine value for butterfly calculation
var aliasReductionCS = [8]float64{
	0.857492925712,
	0.881741997318,
	0.949628649103,
	0.983314592492,
	0.995517816065,
	0.999160558175,
	0.999899195243,
	0.999993155067,
}

// sine value for butterfly calculation
var aliasReductionCA = [8]float64{
	-0.514495755427,
	-0.471731968565,
	-0.313377454204,
	-0.181913199611,
	-0.0945741925262,
	-0.0409655828853,
	-0.0141985685725,
	-0.00369997467375,
}

func ApplyAliasReduction(gc *common.GranuleChannelInfo, values []float64) error {
	if gc == nil {
		return fmt.Errorf("nil granule channel info")
	}
	if len(values) != 576 {
		return fmt.Errorf("alias reduction requires 576 spectral lines: got %d", len(values))
	}

	if gc.GetWindowSwitching() { // no need to apply alias reduction for short blocks
		return nil
	}

	sblim := 32
	if gc.GetMixedBlockFlag() {
		sblim = 2 // only the first 2 subbands have aliasing in mixed blocks, the rest are short blocks
	}

	for sb := 1; sb < sblim; sb++ {
		base := sb * 18
		for i := range len(aliasReductionCS) {
			li := base - 1 - i // lower index moving to lower frequencies
			ui := base + i     // upper index moving to higher frequencies
			lower := values[li]
			upper := values[ui]
			// butterfly calculation
			// (values[li]) = (cs[i], -ca[i])(values[li])
			//  values[ui]     ca[i], cs[i]   values[ui]
			values[li] = lower*aliasReductionCS[i] - upper*aliasReductionCA[i]
			values[ui] = upper*aliasReductionCS[i] + lower*aliasReductionCA[i]
		}
	}

	return nil
}
