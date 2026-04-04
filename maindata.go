package byrd

import (
	"fmt"
)

const RESERVOIR_MAX = 511 // 2^9 - 1 which is the size of main_data_begin field of side info

const (
	SCALEFACTOR_LONG_SFB_0_START = 0
	SCALEFACTOR_LONG_SFB_0_END   = 5
	SCALEFACTOR_LONG_SFB_1_START = 6
	SCALEFACTOR_LONG_SFB_1_END   = 10
	SCALEFACTOR_LONG_SFB_2_START = 11
	SCALEFACTOR_LONG_SFB_2_END   = 15
	SCALEFACTOR_LONG_SFB_3_START = 16
	SCALEFACTOR_LONG_SFB_3_END   = 20

	SCALEFACTOR_MIXED_LONG_START      = 0
	SCALEFACTOR_MIXED_LONG_END        = 7
	SCALEFACTOR_MIXED_SHORT_LOW_START = 3
	SCALEFACTOR_MIXED_SHORT_LOW_END   = 5

	SCALEFACTOR_SHORT_LOW_START  = 0
	SCALEFACTOR_SHORT_LOW_END    = 5
	SCALEFACTOR_SHORT_HIGH_START = 6
	SCALEFACTOR_SHORT_HIGH_END   = 11

	SCALEFACTOR_SHORT_WINDOW_COUNT = 3
)

// generate main data from reservoir offset
func ReadMainData(mainDataBegin uint16, mainDataReservoir *[]byte, cur []byte, mainData *[]byte) error {
	if mainDataReservoir == nil {
		return fmt.Errorf("nil main data reservoir")
	}
	if mainData == nil {
		return fmt.Errorf("nil main data buffer")
	}
	// mainDataBegin is the reverse offset from the end of the reservoir, so it can't be larger than the reservoir itself
	if int(mainDataBegin) > len(*mainDataReservoir) {
		return fmt.Errorf("bit reservoir underflow: need %d bytes, have %d", mainDataBegin, len(*mainDataReservoir))
	}
	start := len(*mainDataReservoir) - int(mainDataBegin)
	mainDataLen := int(mainDataBegin) + len(cur)

	// we reuse mainData buffer for reducing GC overhead, but grow it if needed
	if cap(*mainData) < mainDataLen {
		*mainData = make([]byte, 0, mainDataLen)
	}
	*mainData = (*mainData)[:0]
	*mainData = append(*mainData, (*mainDataReservoir)[start:]...) // append the last mainDataBegin bytes from reservoir
	*mainData = append(*mainData, cur...)                          // append the current frame's main data
	// update reservoir for next frame
	*mainDataReservoir = append(*mainDataReservoir, cur...)
	if len(*mainDataReservoir) > RESERVOIR_MAX { // only have to keep RESERVOIR_MAX bytes, so truncate the buffer
		*mainDataReservoir = (*mainDataReservoir)[len(*mainDataReservoir)-RESERVOIR_MAX:]
	}

	return nil
}

type Scalefactors struct {
	Long  [21]uint8    // 21 bands (11 for slen1 and 10 for slen2) for long blocks
	Short [12][3]uint8 // 12 bands for short blocks, each with 3 windows
}

func readScalefactorBits(br *BitReader, limit int, width int, scratch *uint32) (uint8, error) {
	if width == 0 {
		return 0, nil
	}
	if br.pos+width > limit {
		return 0, fmt.Errorf("scalefactors exceed part23 length: need %d more bits, have %d", width, limit-br.pos)
	}
	if err := br.ReadBitsTo(scratch, width); err != nil {
		return 0, err
	}
	return uint8(*scratch), nil
}

func readLongScalefactorRange(br *BitReader, limit int, width int, from int, to int, dst *Scalefactors, scratch *uint32) error {
	for sfb := from; sfb <= to; sfb++ {
		v, err := readScalefactorBits(br, limit, width, scratch)
		if err != nil {
			return err
		}
		dst.Long[sfb] = v
	}
	return nil
}

func readShortScalefactorRange(br *BitReader, limit int, width int, from int, to int, dst *Scalefactors, scratch *uint32) error {
	for sfb := from; sfb <= to; sfb++ {
		for win := range SCALEFACTOR_SHORT_WINDOW_COUNT {
			v, err := readScalefactorBits(br, limit, width, scratch)
			if err != nil {
				return err
			}
			dst.Short[sfb][win] = v
		}
	}
	return nil
}

func ParseScaleFactor(br *BitReader, gc *GranuleChannelInfo, scfsi [4]byte, granule int, prev *Scalefactors, scaleFactors *Scalefactors) (int, error) {
	if br == nil {
		return 0, fmt.Errorf("nil BitReader")
	}
	if gc == nil {
		return 0, fmt.Errorf("nil granule channel info")
	}
	if scaleFactors == nil {
		return 0, fmt.Errorf("nil scalefactors")
	}
	if granule < 0 || granule > 1 {
		return 0, fmt.Errorf("invalid granule index: %d", granule)
	}
	if int(gc.ScalefacCompress) >= len(SCALEFACTOR_COMPRESS) {
		return 0, fmt.Errorf("invalid scalefactor_compress: %d", gc.ScalefacCompress)
	}

	*scaleFactors = Scalefactors{}
	start := br.pos
	limit := start + int(gc.Part23Length)
	slen := SCALEFACTOR_COMPRESS[gc.ScalefacCompress]
	var scratch uint32

	longSyntax := !gc.GetWindowSwitching() || BlockType(gc.GetBlockType()) != BlockTypeShort
	switch {
	case longSyntax:
		groups := [...]struct {
			from, to int
			width    int
		}{
			{SCALEFACTOR_LONG_SFB_0_START, SCALEFACTOR_LONG_SFB_0_END, slen.Slen1},
			{SCALEFACTOR_LONG_SFB_1_START, SCALEFACTOR_LONG_SFB_1_END, slen.Slen1},
			{SCALEFACTOR_LONG_SFB_2_START, SCALEFACTOR_LONG_SFB_2_END, slen.Slen2},
			{SCALEFACTOR_LONG_SFB_3_START, SCALEFACTOR_LONG_SFB_3_END, slen.Slen2},
		}
		for i, group := range groups {
			if granule == 1 && scfsi[i] == 1 {
				if prev == nil {
					return 0, fmt.Errorf("missing previous granule scalefactors for scfsi reuse")
				}
				copy(scaleFactors.Long[group.from:group.to+1], prev.Long[group.from:group.to+1])
				continue
			}
			if err := readLongScalefactorRange(br, limit, group.width, group.from, group.to, scaleFactors, &scratch); err != nil {
				return 0, err
			}
		}
	case gc.GetMixedBlockFlag():
		if err := readLongScalefactorRange(br, limit, slen.Slen1, SCALEFACTOR_MIXED_LONG_START, SCALEFACTOR_MIXED_LONG_END, scaleFactors, &scratch); err != nil {
			return 0, err
		}
		if err := readShortScalefactorRange(br, limit, slen.Slen1, SCALEFACTOR_MIXED_SHORT_LOW_START, SCALEFACTOR_MIXED_SHORT_LOW_END, scaleFactors, &scratch); err != nil {
			return 0, err
		}
		if err := readShortScalefactorRange(br, limit, slen.Slen2, SCALEFACTOR_SHORT_HIGH_START, SCALEFACTOR_SHORT_HIGH_END, scaleFactors, &scratch); err != nil {
			return 0, err
		}
	default:
		if err := readShortScalefactorRange(br, limit, slen.Slen1, SCALEFACTOR_SHORT_LOW_START, SCALEFACTOR_SHORT_LOW_END, scaleFactors, &scratch); err != nil {
			return 0, err
		}
		if err := readShortScalefactorRange(br, limit, slen.Slen2, SCALEFACTOR_SHORT_HIGH_START, SCALEFACTOR_SHORT_HIGH_END, scaleFactors, &scratch); err != nil {
			return 0, err
		}
	}

	return br.pos - start, nil
}

func selectTable(sampleRate uint16, gc *GranuleChannelInfo, spectralLineIndex int) (*HuffmanTable, error) {
	if gc == nil {
		return nil, fmt.Errorf("nil granule channel info")
	}
	if spectralLineIndex < 0 {
		return nil, fmt.Errorf("invalid spectral line index: %d", spectralLineIndex)
	}

	sfBands, ok := SCALEFACTOR_BAND_INDICES[sampleRate] // sfb depends on sample rate
	if !ok {
		return nil, fmt.Errorf("unsupported sample rate for scalefactor bands: %d", sampleRate)
	}

	var tableIndex int
	longSyntax := !gc.GetWindowSwitching() || gc.GetBlockType() != BlockTypeShort
	if longSyntax {
		region1StartSFB := int(gc.Region0Count) + 1
		region2StartSFB := int(gc.Region0Count) + int(gc.Region1Count) + 2
		if region1StartSFB >= len(sfBands.Long) {
			region1StartSFB = len(sfBands.Long) - 1
		}
		if region2StartSFB >= len(sfBands.Long) {
			region2StartSFB = len(sfBands.Long) - 1
		}
		region1Start := sfBands.Long[region1StartSFB]
		region2Start := sfBands.Long[region2StartSFB]
		if spectralLineIndex < region1Start {
			tableIndex = int(gc.TableSelect[0])
		} else if spectralLineIndex < region2Start {
			tableIndex = int(gc.TableSelect[1])
		} else {
			tableIndex = int(gc.TableSelect[2])
		}
	} else {
		region1Start := sfBands.Short[3] * SCALEFACTOR_SHORT_WINDOW_COUNT
		if gc.GetMixedBlockFlag() {
			region1Start = sfBands.Long[8]
		}
		if spectralLineIndex < region1Start {
			tableIndex = int(gc.TableSelect[0])
		} else {
			tableIndex = int(gc.TableSelect[1])
		}
	}

	table, ok := baseTables[tableIndex]
	if !ok {
		return nil, fmt.Errorf("unknown huffman table: %d", tableIndex)
	}
	if table.Data == nil {
		return nil, fmt.Errorf("unsupported huffman table: %d", tableIndex)
	}
	return &table, nil
}

func ParseBigValues(br *BitReader, gc *GranuleChannelInfo, mainData []byte, scalefactors *Scalefactors) ([]byte, error) {
	return nil, fmt.Errorf("ParseBigValues not implemented")
}
