package byrd

import (
	"bufio"
	"fmt"
	"io"
)

const RESERVOIR_MAX = 511 // 2^9 - 1 which is the size of main_data_begin field of side info

// only reads main data, does not parse it
func ReadMainData(r *bufio.Reader, mainDataBegin uint16, mainDataLen int, mainDataReservoir *[]byte) ([]byte, error) {
	cur := make([]byte, mainDataLen)
	_, err := io.ReadFull(r, cur)
	if err != nil {
		return nil, err
	}
	// mainDataBegin is the reverse offset from the end of the reservoir, so it can't be larger than the reservoir itself
	if int(mainDataBegin) > len(*mainDataReservoir) {
		return nil, fmt.Errorf("bit reservoir underflow: need %d bytes, have %d", mainDataBegin, len(*mainDataReservoir))
	}
	start := len(*mainDataReservoir) - int(mainDataBegin)

	mainData := make([]byte, 0, int(mainDataBegin)+mainDataLen)
	mainData = append(mainData, (*mainDataReservoir)[start:]...) // append the last mainDataBegin bytes from reservoir
	mainData = append(mainData, cur...)                          // append the current frame's main data to the end of main data
	// update reservoir for next frame
	*mainDataReservoir = append(*mainDataReservoir, cur...)
	if len(*mainDataReservoir) > RESERVOIR_MAX { // only have to keep RESERVOIR_MAX bytes, so truncate the buffer
		*mainDataReservoir = (*mainDataReservoir)[len(*mainDataReservoir)-RESERVOIR_MAX:]
	}

	return mainData, nil
}
