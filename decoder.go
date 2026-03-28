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

// read single MP3 frame
func DecodeMP3Frame(r *bufio.Reader) {
	var h MP3FrameHeader
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
	_, err = ReadMainData(r, sideInfo.MainDataBegin, mainDataLen)
	if err != nil {
		fmt.Printf("failed to read main data: %v\n", err)
		return
	}
}
