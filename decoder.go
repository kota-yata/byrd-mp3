package byrd

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

const GRANULE_COUNT = 2

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
	var scalefactors [2][2]Scalefactors
	var spectralValues [2][2][]int
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
		err = ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, &mainData)
		if err != nil {
			fmt.Printf("failed to read main data: %v\n", err)
			return
		}
		br := NewBitReader(mainData)
		channels := 2
		if h.GetChannelMode() == ChannelModeMono {
			channels = 1
		}
		for gr := range GRANULE_COUNT {
			for ch := 0; ch < channels; ch++ {
				gc := &sideInfo.Granule[gr][ch]
				part23Start := br.pos
				part23End := part23Start + int(gc.Part23Length)

				var prev *Scalefactors
				if gr == 1 {
					prev = &scalefactors[0][ch]
				}
				_, err = ParseScaleFactor(br, gc, sideInfo.SCFSI[ch], gr, prev, &scalefactors[gr][ch])
				if err != nil {
					fmt.Printf("failed to parse scalefactors: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}

				huffmanLen := part23End - br.pos
				if huffmanLen < 0 {
					fmt.Printf("main data underrun: frame granule=%d channel=%d part23=%d bits consumed for scalefactors=%d\n", gr, ch, gc.Part23Length, br.pos-part23Start)
					return
				}
				_, err = ParseBigValues(br, h.GetSampleRate(), gc, part23End, &spectralValues[gr][ch])
				if err != nil {
					fmt.Printf("failed to parse big values: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}

				// TODO: Implement huffman part parser. Skip the remaining part3 bits here until Huffman decoding is implemented.
				br.pos = part23End
				if br.pos > len(mainData)*8 {
					fmt.Printf("main data overrun: frame granule=%d channel=%d part23=%d\n", gr, ch, gc.Part23Length)
					return
				}
			}
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
