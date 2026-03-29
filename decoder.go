package byrd

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func OpenMP3File(path string) (io.ReadCloser, error) {
	ext := ".mp3"
	if len(path) < len(ext) || path[len(path)-len(ext):] != ext {
		return nil, fmt.Errorf("unsupported file format: %s", path)
	}
	return os.Open(path)
}

func DecodeMP3Frames(r *bufio.Reader) {
	var h MP3FrameHeader
	var mainDataReservoir []byte
	var cur []byte
	var mainData []byte
	for {
		h = MP3FrameHeader{} // reset frame state
		if err := ReadHeader(&h, r); err != nil {
			fmt.Printf("failed to read MP3 frame header: %v\n", err)
			return
		}

		if !h.ValidateCRC(r) {
			fmt.Printf("CRC check failed for MP3 frame\n")
			return
		}

		sideInfoLen := GetSideInfoLength(&h)
		sideInfo, err := ReadSideInfo(&h, r, sideInfoLen)
		if err != nil {
			fmt.Printf("failed to read side info: %v\n", err)
			return
		}

		frameLen, err := h.GetFrameLength()
		if err != nil {
			fmt.Printf("failed to calculate frame length: %v\n", err)
			return
		}
		crcLen := 0
		if h.HasCRC() {
			crcLen = 2
		}

		mainDataLen := frameLen - 4 - sideInfoLen - crcLen
		// we reuse cur buffer for reducing GC overhead, but grow it if needed
		if cap(cur) < mainDataLen {
			cur = make([]byte, mainDataLen)
		}
		cur = cur[:mainDataLen]
		_, err = io.ReadFull(r, cur)
		if err != nil {
			fmt.Printf("failed to read main data: %v\n", err)
			return
		}
		mainData, err = ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, mainData)
		if err != nil {
			fmt.Printf("failed to read main data: %v\n", err)
			return
		}

		// check stream end
		if _, err := r.Peek(1); err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("failed to check next frame: %v\n", err)
			return
		}
	}
}
