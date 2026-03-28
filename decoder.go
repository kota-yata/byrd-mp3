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
		// handle error
		return
	}

	if !ValidateCRC(&h, r) {
		// handle CRC validation failure
		return
	}

	_, err := ReadSideInfo(&h, r)
	if err != nil {
		// handle error
		return
	}

}
