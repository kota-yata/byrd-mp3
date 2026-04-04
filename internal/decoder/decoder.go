package decoder

import (
	"bufio"
	"byrd/internal/common"
	"byrd/internal/header"
	"byrd/internal/hybrid"
	"byrd/internal/maindata"
	"byrd/internal/stereo"
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
	var h header.MP3FrameHeader
	var mainDataReservoir []byte
	var cur []byte
	var mainData []byte
	var scalefactors [2][2]maindata.Scalefactors
	var spectralValues [2][2][576]int
	var requantizedValues [2][2][576]float64
	var reorderedValues [2][2][576]float64
	var hybridValues [2][2][576]float64
	for {
		h = header.MP3FrameHeader{} // reset frame state
		if err := header.ReadHeader(&h, r); err != nil {
			fmt.Printf("failed to read MP3 frame header: %v\n", err)
			return
		}

		if !h.ValidateCRC(r) {
			fmt.Printf("CRC check failed for MP3 frame\n")
			return
		}

		sideInfoLen := header.GetSideInfoLength(&h)
		sideInfo, err := header.ReadSideInfo(&h, r, sideInfoLen)
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
		err = maindata.ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, &mainData)
		if err != nil {
			fmt.Printf("failed to read main data: %v\n", err)
			return
		}
		br := common.NewBitReader(mainData)
		channels := 2
		if h.GetChannelMode() == header.ChannelModeMono {
			channels = 1
		}
		for gr := range GRANULE_COUNT {
			for ch := 0; ch < channels; ch++ {
				gc := &sideInfo.Granule[gr][ch]
				part23Start := br.Pos
				part23End := part23Start + int(gc.Part23Length)

				var prev *maindata.Scalefactors
				if gr == 1 {
					prev = &scalefactors[0][ch]
				}
				_, err = maindata.ParseScaleFactor(br, gc, sideInfo.SCFSI[ch], gr, prev, &scalefactors[gr][ch])
				if err != nil {
					fmt.Printf("failed to parse scalefactors: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}

				huffmanLen := part23End - br.Pos
				if huffmanLen < 0 {
					fmt.Printf("main data underrun: frame granule=%d channel=%d part23=%d bits consumed for scalefactors=%d\n", gr, ch, gc.Part23Length, br.Pos-part23Start)
					return
				}
				spectralBuffer := spectralValues[gr][ch][:]
				_, err = maindata.ParseBigValues(br, h.GetSampleRate(), gc, part23End, &spectralBuffer)
				if err != nil {
					fmt.Printf("failed to parse big values: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}
				_, err = maindata.ParseCount1Values(br, gc, part23End, &spectralBuffer)
				if err != nil {
					fmt.Printf("failed to parse count1 values: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}
				requantizedBuffer := requantizedValues[gr][ch][:]
				if err := maindata.Requantize(h.GetSampleRate(), gc, &scalefactors[gr][ch], spectralBuffer, &requantizedBuffer); err != nil {
					fmt.Printf("failed to requantize values: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}
				reorderedBuffer := reorderedValues[gr][ch][:]
				if err := maindata.Reorder(h.GetSampleRate(), gc, requantizedBuffer, &reorderedBuffer); err != nil {
					fmt.Printf("failed to reorder values: frame granule=%d channel=%d err=%v\n", gr, ch, err)
					return
				}
				br.Pos = part23End
				if br.Pos > len(mainData)*8 {
					fmt.Printf("main data overrun: frame granule=%d channel=%d part23=%d\n", gr, ch, gc.Part23Length)
					return
				}
			}
			if channels == 2 {
				left := reorderedValues[gr][0][:]
				right := reorderedValues[gr][1][:]
				if err := stereo.ApplyJointStereo(h.GetChannelMode(), h.GetModeExtension(), left, right); err != nil {
					fmt.Printf("failed to apply joint stereo: frame granule=%d err=%v\n", gr, err)
					return
				}
			}
			for ch := 0; ch < channels; ch++ {
				hybridBuffer := hybridValues[gr][ch][:]
				copy(hybridBuffer, reorderedValues[gr][ch][:])
				if err := hybrid.ApplyAliasReduction(&sideInfo.Granule[gr][ch], hybridBuffer); err != nil {
					fmt.Printf("failed to apply alias reduction: frame granule=%d channel=%d err=%v\n", gr, ch, err)
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
