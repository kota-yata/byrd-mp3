package decoder

import (
	"bufio"
	"byrd/internal/common"
	"byrd/internal/header"
	"byrd/internal/hybrid"
	"byrd/internal/maindata"
	"byrd/internal/stereo"
	"byrd/internal/synthesis"
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

func DecodeMP3Frames(r *bufio.Reader) ([]int16, uint16, int, error) {
	var h header.MP3FrameHeader
	var mainDataReservoir []byte
	var cur []byte
	var mainData []byte
	var scalefactors [2][2]maindata.Scalefactors
	var count1 [2][2]int
	var spectralValues [2][2][576]int
	var requantizedValues [2][2][576]float64
	var reorderedValues [2][2][576]float64
	var hybridValues [2][2][576]float64
	var overlapState [2][32][18]float64
	var hybridSamples [2][2][32][18]float64
	var synthesisState [2]synthesis.PolyphaseState
	var pcmSamples [2][2][576]float64
	var out []int16
	var sampleRate uint16
	channels := 0
	for {
		h = header.MP3FrameHeader{} // reset frame state
		if err := header.ReadHeader(&h, r); err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, 0, fmt.Errorf("failed to read MP3 frame header: %w", err)
		}

		if !h.ValidateCRC(r) {
			return nil, 0, 0, fmt.Errorf("CRC check failed for MP3 frame")
		}

		sideInfoLen := header.GetSideInfoLength(&h)
		sideInfo, err := header.ReadSideInfo(&h, r, sideInfoLen)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to read side info: %w", err)
		}

		frameLen, err := h.GetFrameLength()
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to calculate frame length: %w", err)
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
			return nil, 0, 0, fmt.Errorf("failed to read main data: %w", err)
		}
		err = maindata.ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, &mainData)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to read main data: %w", err)
		}
		br := common.NewBitReader(mainData)
		frameChannels := 2
		if h.GetChannelMode() == header.ChannelModeMono {
			frameChannels = 1
		}
		if sampleRate == 0 {
			sampleRate = h.GetSampleRate()
			channels = frameChannels
		} else if sampleRate != h.GetSampleRate() || channels != frameChannels {
			return nil, 0, 0, fmt.Errorf("variable stream parameters are not supported: sampleRate=%d/%d channels=%d/%d", sampleRate, h.GetSampleRate(), channels, frameChannels)
		}
		for gr := range GRANULE_COUNT {
			for ch := 0; ch < frameChannels; ch++ {
				gc := &sideInfo.Granule[gr][ch]
				part23Start := br.Pos
				part23End := part23Start + int(gc.Part23Length)

				var prev *maindata.Scalefactors
				if gr == 1 {
					prev = &scalefactors[0][ch]
				}
				_, err = maindata.ParseScaleFactor(br, gc, sideInfo.SCFSI[ch], gr, prev, &scalefactors[gr][ch])
				if err != nil {
					return nil, 0, 0, fmt.Errorf("failed to parse scalefactors: frame granule=%d channel=%d err=%w", gr, ch, err)
				}

				huffmanLen := part23End - br.Pos
				if huffmanLen < 0 {
					return nil, 0, 0, fmt.Errorf("main data underrun: frame granule=%d channel=%d part23=%d bits consumed for scalefactors=%d", gr, ch, gc.Part23Length, br.Pos-part23Start)
				}
				spectralBuffer := spectralValues[gr][ch][:]
				_, err = maindata.ParseBigValues(br, h.GetSampleRate(), gc, part23End, &spectralBuffer)
				if err != nil {
					return nil, 0, 0, fmt.Errorf("failed to parse big values: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
				count1Lines, err := maindata.ParseCount1Values(br, gc, part23End, &spectralBuffer)
				if err != nil {
					return nil, 0, 0, fmt.Errorf("failed to parse count1 values: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
				count1[gr][ch] = int(gc.BigValues)*2 + count1Lines
				requantizedBuffer := requantizedValues[gr][ch][:]
				if err := maindata.Requantize(h.GetSampleRate(), gc, &scalefactors[gr][ch], spectralBuffer, &requantizedBuffer); err != nil {
					return nil, 0, 0, fmt.Errorf("failed to requantize values: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
				reorderedBuffer := reorderedValues[gr][ch][:]
				if err := maindata.Reorder(h.GetSampleRate(), gc, requantizedBuffer, &reorderedBuffer); err != nil {
					return nil, 0, 0, fmt.Errorf("failed to reorder values: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
				br.Pos = part23End
				if br.Pos > len(mainData)*8 {
					return nil, 0, 0, fmt.Errorf("main data overrun: frame granule=%d channel=%d part23=%d", gr, ch, gc.Part23Length)
				}
			}
			if frameChannels == 2 {
				left := reorderedValues[gr][0][:]
				right := reorderedValues[gr][1][:]
				if err := stereo.ApplyJointStereo(h.GetSampleRate(), h.GetChannelMode(), h.GetModeExtension(), &sideInfo.Granule[gr][0], &scalefactors[gr][0], left, right, count1[gr][0], count1[gr][1]); err != nil {
					return nil, 0, 0, fmt.Errorf("failed to apply joint stereo: frame granule=%d err=%w", gr, err)
				}
			}
			for ch := 0; ch < frameChannels; ch++ {
				hybridBuffer := hybridValues[gr][ch][:]
				copy(hybridBuffer, reorderedValues[gr][ch][:])
				if err := hybrid.ApplyAliasReduction(&sideInfo.Granule[gr][ch], hybridBuffer); err != nil {
					return nil, 0, 0, fmt.Errorf("failed to apply alias reduction: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
				if err := hybrid.HybridSynthesis(&sideInfo.Granule[gr][ch], hybridBuffer, &overlapState[ch], &hybridSamples[gr][ch]); err != nil {
					return nil, 0, 0, fmt.Errorf("failed to run hybrid synthesis: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
				synthesis.ApplyFrequencyInversion(&hybridSamples[gr][ch])
				if err := synthesis.SynthesizeGranule(&hybridSamples[gr][ch], &synthesisState[ch], &pcmSamples[gr][ch]); err != nil {
					return nil, 0, 0, fmt.Errorf("failed to run polyphase synthesis: frame granule=%d channel=%d err=%w", gr, ch, err)
				}
			}
			if frameChannels == 1 {
				for i := 0; i < 576; i++ {
					out = append(out, synthesis.QuantizeSample(pcmSamples[gr][0][i]))
				}
			} else {
				for i := 0; i < 576; i++ {
					out = append(out,
						synthesis.QuantizeSample(pcmSamples[gr][0][i]),
						synthesis.QuantizeSample(pcmSamples[gr][1][i]),
					)
				}
			}
		}
	}

	return out, sampleRate, channels, nil
}
