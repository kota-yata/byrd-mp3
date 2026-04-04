package decoder

import (
	"bufio"
	"byrd/internal/common"
	"byrd/internal/header"
	"byrd/internal/maindata"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// Just to make sure no error occurs when parsing real MP3 data
func TestParseOutputMP3RealData(t *testing.T) {
	f, err := OpenMP3File(filepath.Join("..", "..", "output.mp3"))
	if err != nil {
		t.Fatalf("failed to open output.mp3: %v", err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var mainDataReservoir []byte
	var cur []byte
	var mainData []byte
	frameIndex := 0

	for {
		var h header.MP3FrameHeader
		err = header.ReadHeader(&h, r)
		if err == io.EOF {
			break
		}
		if err != nil {
			if frameIndex > 0 {
				t.Logf("stopping after %d parsed frames: %v", frameIndex, err)
				break
			}
			t.Fatalf("frame %d: failed to read header: %v", frameIndex, err)
		}

		bitrateKbps, free := h.GetBitrateKbps()
		if free {
			t.Fatalf("frame %d: free bitrate is not supported", frameIndex)
		}
		frameLen, err := h.GetFrameLength()
		if err != nil {
			t.Fatalf("frame %d: failed to get frame length: %v", frameIndex, err)
		}
		if !h.ValidateCRC(r) {
			t.Fatalf("frame %d: CRC validation failed", frameIndex)
		}

		sideInfoLen := header.GetSideInfoLength(&h)
		sideInfo, err := header.ReadSideInfo(&h, r, sideInfoLen)
		if err != nil {
			t.Fatalf("frame %d: failed to read side info: %v", frameIndex, err)
		}

		crcLen := 0
		if h.HasCRC() {
			crcLen = 2
		}
		mainDataLen := frameLen - 4 - sideInfoLen - crcLen
		if mainDataLen < 0 {
			t.Fatalf("frame %d: invalid main data length %d", frameIndex, mainDataLen)
		}

		if cap(cur) < mainDataLen {
			cur = make([]byte, mainDataLen)
		}
		cur = cur[:mainDataLen]
		_, err = io.ReadFull(r, cur)
		if err != nil {
			t.Fatalf("frame %d: failed to read current frame main data: %v", frameIndex, err)
		}
		err = maindata.ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, &mainData)
		if err != nil {
			t.Fatalf("frame %d: failed to reconstruct main data: %v", frameIndex, err)
		}

		channels := 2
		if h.GetChannelMode() == header.ChannelModeMono {
			channels = 1
		}
		frameSummary := []string{
			fmt.Sprintf(
				"bitrate=%dkbps sampleRate=%d padding=%v hasCRC=%v channelMode=%s modeExt=%d copyright=%v original=%v emphasis=%d frameLen=%d sideInfoLen=%d mainDataBegin=%d mainDataLen=%d reservoirLen=%d",
				bitrateKbps,
				h.GetSampleRate(),
				h.Padding(),
				h.HasCRC(),
				h.GetChannelMode(),
				h.GetModeExtension(),
				h.IsCopyrighted(),
				h.IsOriginal(),
				h.GetEmphasis(),
				frameLen,
				sideInfoLen,
				sideInfo.MainDataBegin,
				mainDataLen,
				len(mainDataReservoir),
			),
		}
		for ch := 0; ch < channels; ch++ {
			frameSummary = append(frameSummary, fmt.Sprintf("ch=%d scfsi=%v", ch, sideInfo.SCFSI[ch]))
		}

		br := common.NewBitReader(mainData)
		var prev [2]maindata.Scalefactors
		var spectralValues [2][576]int
		var requantizedValues [2][576]float64
		for gr := 0; gr < 2; gr++ {
			for ch := 0; ch < channels; ch++ {
				gc := &sideInfo.Granule[gr][ch]
				part23Start := br.Pos
				part23End := part23Start + int(gc.Part23Length)
				var scalefactors maindata.Scalefactors
				var prevPtr *maindata.Scalefactors
				if gr == 1 {
					prevPtr = &prev[ch]
				}

				part2Bits, err := maindata.ParseScaleFactor(br, gc, sideInfo.SCFSI[ch], gr, prevPtr, &scalefactors)
				if err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to parse scalefactors: %v", frameIndex, gr, ch, err)
				}
				prev[ch] = scalefactors

				spectralBuffer := spectralValues[ch][:]
				bigValueLines, err := maindata.ParseBigValues(br, h.GetSampleRate(), gc, part23End, &spectralBuffer)
				if err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to parse big values: %v", frameIndex, gr, ch, err)
				}
				count1Lines, err := maindata.ParseCount1Values(br, gc, part23End, &spectralBuffer)
				if err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to parse count1 values: %v", frameIndex, gr, ch, err)
				}
				requantizedBuffer := requantizedValues[ch][:]
				if err := maindata.Requantize(h.GetSampleRate(), gc, &scalefactors, spectralBuffer, &requantizedBuffer); err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to requantize values: %v", frameIndex, gr, ch, err)
				}
				nonZeroRequantized := 0
				for _, v := range requantizedBuffer {
					if v != 0 {
						nonZeroRequantized++
					}
				}
				frameSummary = append(frameSummary, fmt.Sprintf(
					"gr=%d ch=%d part23=%d part2=%d part3=%d bigValues=%d bigValueLines=%d count1Lines=%d globalGain=%d scalefacCompress=%d tableSelect=%v subblockGain=%v region0=%d region1=%d windowSwitching=%v blockType=%s mixed=%v preflag=%v scalefacScale=%v count1Table=%v long=%v short=%v spectralLines=%d requantizedNonZero=%d",
					gr,
					ch,
					gc.Part23Length,
					part2Bits,
					int(gc.Part23Length)-part2Bits,
					gc.BigValues,
					bigValueLines,
					count1Lines,
					gc.GlobalGain,
					gc.ScalefacCompress,
					gc.TableSelect,
					gc.SubblockGain,
					gc.Region0Count,
					gc.Region1Count,
					gc.GetWindowSwitching(),
					gc.GetBlockType(),
					gc.GetMixedBlockFlag(),
					gc.GetPreflag(),
					gc.GetScalefacScale(),
					gc.GetCount1TableSelect(),
					scalefactors.Long,
					scalefactors.Short,
					576,
					nonZeroRequantized,
				))

				br.Pos = part23End
				if br.Pos > len(mainData)*8 {
					t.Fatalf("frame %d gr=%d ch=%d: part23 overruns main data bitstream", frameIndex, gr, ch)
				}
			}
		}

		t.Logf("frame=%d %s", frameIndex, strings.Join(frameSummary, " | "))
		frameIndex++
	}

	if frameIndex == 0 {
		t.Fatalf("no frames parsed from output.mp3")
	}
}
