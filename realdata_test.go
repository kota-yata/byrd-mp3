package byrd

import (
	"bufio"
	"io"
	"testing"
)

// Just to make sure no error occurs when parsing real MP3 data
func TestParseOutputMP3RealData(t *testing.T) {
	f, err := OpenMP3File("output.mp3")
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
		var h MP3FrameHeader
		err = ReadHeader(&h, r)
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

		sideInfoLen := GetSideInfoLength(&h)
		sideInfo, err := ReadSideInfo(&h, r, sideInfoLen)
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
		err = ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, &mainData)
		if err != nil {
			t.Fatalf("frame %d: failed to reconstruct main data: %v", frameIndex, err)
		}

		t.Logf(
			"frame=%d bitrate=%dkbps sampleRate=%d padding=%v hasCRC=%v channelMode=%s modeExt=%d copyright=%v original=%v emphasis=%d frameLen=%d sideInfoLen=%d mainDataBegin=%d mainDataLen=%d reservoirLen=%d",
			frameIndex,
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
		)

		channels := 2
		if h.GetChannelMode() == ChannelModeMono {
			channels = 1
		}
		for ch := 0; ch < channels; ch++ {
			t.Logf("frame=%d ch=%d scfsi=%v", frameIndex, ch, sideInfo.SCFSI[ch])
		}

		br := NewBitReader(mainData)
		var prev [2]Scalefactors
		var spectralValues [2][]int
		for gr := 0; gr < 2; gr++ {
			for ch := 0; ch < channels; ch++ {
				gc := &sideInfo.Granule[gr][ch]
				part23Start := br.pos
				part23End := part23Start + int(gc.Part23Length)
				var scalefactors Scalefactors
				var prevPtr *Scalefactors
				if gr == 1 {
					prevPtr = &prev[ch]
				}

				part2Bits, err := ParseScaleFactor(br, gc, sideInfo.SCFSI[ch], gr, prevPtr, &scalefactors)
				if err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to parse scalefactors: %v", frameIndex, gr, ch, err)
				}
				prev[ch] = scalefactors

				bigValueLines, err := ParseBigValues(br, h.GetSampleRate(), gc, part23End, &spectralValues[ch])
				if err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to parse big values: %v", frameIndex, gr, ch, err)
				}
				count1Lines, err := ParseCount1Values(br, gc, part23End, &spectralValues[ch])
				if err != nil {
					t.Fatalf("frame %d gr=%d ch=%d: failed to parse count1 values: %v", frameIndex, gr, ch, err)
				}

				t.Logf(
					"frame=%d gr=%d ch=%d part23=%d part2=%d part3=%d bigValues=%d bigValueLines=%d count1Lines=%d globalGain=%d scalefacCompress=%d tableSelect=%v subblockGain=%v region0=%d region1=%d windowSwitching=%v blockType=%s mixed=%v preflag=%v scalefacScale=%v count1Table=%v long=%v short=%v spectral=%v",
					frameIndex,
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
					spectralValues[ch],
				)

				br.pos = part23End
				if br.pos > len(mainData)*8 {
					t.Fatalf("frame %d gr=%d ch=%d: part23 overruns main data bitstream", frameIndex, gr, ch)
				}
			}
		}

		frameIndex++
	}

	if frameIndex == 0 {
		t.Fatalf("no frames parsed from output.mp3")
	}
}
