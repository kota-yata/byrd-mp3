package byrd

import (
	"fmt"
)

const RESERVOIR_MAX = 511 // 2^9 - 1 which is the size of main_data_begin field of side info

// generate main data from reservoir offset
func ReadMainData(mainDataBegin uint16, mainDataReservoir *[]byte, cur []byte, mainData []byte) ([]byte, error) {
	// mainDataBegin is the reverse offset from the end of the reservoir, so it can't be larger than the reservoir itself
	if int(mainDataBegin) > len(*mainDataReservoir) {
		return nil, fmt.Errorf("bit reservoir underflow: need %d bytes, have %d", mainDataBegin, len(*mainDataReservoir))
	}
	start := len(*mainDataReservoir) - int(mainDataBegin)
	mainDataLen := int(mainDataBegin) + len(cur)

	// we reuse mainData buffer for reducing GC overhead, but grow it if needed
	if cap(mainData) < mainDataLen {
		mainData = make([]byte, 0, mainDataLen)
	}
	mainData = mainData[:0]
	mainData = append(mainData, (*mainDataReservoir)[start:]...) // append the last mainDataBegin bytes from reservoir
	mainData = append(mainData, cur...)                          // append the current frame's main data
	// update reservoir for next frame
	*mainDataReservoir = append(*mainDataReservoir, cur...)
	if len(*mainDataReservoir) > RESERVOIR_MAX { // only have to keep RESERVOIR_MAX bytes, so truncate the buffer
		*mainDataReservoir = (*mainDataReservoir)[len(*mainDataReservoir)-RESERVOIR_MAX:]
	}

	return mainData, nil
}

func ParseMainData(mainData []byte, part23Len uint16, scalefactorCompress byte) {

}
