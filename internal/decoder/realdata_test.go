package decoder

import (
	"bufio"
	"byrd/internal/common"
	"byrd/internal/header"
	"byrd/internal/maindata"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Just to make sure no error occurs when parsing bundled MP3 data.
func TestParseStaticMP3RealData(t *testing.T) {
	pattern := filepath.Join("..", "..", "static", "*.mp3")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("failed to list static mp3 files: %v", err)
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		t.Fatalf("no mp3 files found under static/")
	}

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			runParseRealDataTest(t, path)
		})
	}
}

func runParseRealDataTest(t *testing.T, path string) {
	t.Helper()

	f, err := OpenMP3File(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", filepath.Base(path), err)
	}
	defer f.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", filepath.Base(path), err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory", filepath.Base(path))
	}
	if info.Size() == 0 {
		t.Fatalf("%s is empty", filepath.Base(path))
	}

	r := bufio.NewReader(f)
	var mainDataReservoir []byte
	var cur []byte
	var mainData []byte
	frameIndex := 0
	fileLabel := filepath.Base(path)

	for {
		var h header.MP3FrameHeader
		err = header.ReadHeader(&h, r)
		if err == io.EOF {
			break
		}
		if err != nil {
			if frameIndex > 0 {
				t.Logf("file=%s stopping after %d parsed frames: %v", fileLabel, frameIndex, err)
				break
			}
			t.Fatalf("file=%s frame=%d: failed to read header: %v", fileLabel, frameIndex, err)
		}

		bitrateKbps, free := h.GetBitrateKbps()
		if free {
			t.Fatalf("file=%s frame=%d: free bitrate is not supported", fileLabel, frameIndex)
		}
		frameLen, err := h.GetFrameLength()
		if err != nil {
			t.Fatalf("file=%s frame=%d: failed to get frame length: %v", fileLabel, frameIndex, err)
		}
		if !h.ValidateCRC(r) {
			t.Fatalf("file=%s frame=%d: CRC validation failed", fileLabel, frameIndex)
		}

		sideInfoLen := header.GetSideInfoLength(&h)
		sideInfo, err := header.ReadSideInfo(&h, r, sideInfoLen)
		if err != nil {
			t.Fatalf("file=%s frame=%d: failed to read side info: %v", fileLabel, frameIndex, err)
		}

		crcLen := 0
		if h.HasCRC() {
			crcLen = 2
		}
		mainDataLen := frameLen - 4 - sideInfoLen - crcLen
		if mainDataLen < 0 {
			t.Fatalf("file=%s frame=%d: invalid main data length %d", fileLabel, frameIndex, mainDataLen)
		}

		if cap(cur) < mainDataLen {
			cur = make([]byte, mainDataLen)
		}
		cur = cur[:mainDataLen]
		_, err = io.ReadFull(r, cur)
		if err != nil {
			t.Fatalf("file=%s frame=%d: failed to read current frame main data: %v", fileLabel, frameIndex, err)
		}
		err = maindata.ReadMainData(sideInfo.MainDataBegin, &mainDataReservoir, cur, &mainData)
		if err != nil {
			t.Fatalf("file=%s frame=%d: failed to reconstruct main data: %v", fileLabel, frameIndex, err)
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
		var reorderedValues [2][576]float64
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
					t.Fatalf("file=%s frame=%d gr=%d ch=%d: failed to parse scalefactors: %v", fileLabel, frameIndex, gr, ch, err)
				}
				prev[ch] = scalefactors

				spectralBuffer := spectralValues[ch][:]
				bigValueLines, err := maindata.ParseBigValues(br, h.GetSampleRate(), gc, part23End, &spectralBuffer)
				if err != nil {
					t.Fatalf("file=%s frame=%d gr=%d ch=%d: failed to parse big values: %v", fileLabel, frameIndex, gr, ch, err)
				}
				count1Lines, err := maindata.ParseCount1Values(br, gc, part23End, &spectralBuffer)
				if err != nil {
					t.Fatalf("file=%s frame=%d gr=%d ch=%d: failed to parse count1 values: %v", fileLabel, frameIndex, gr, ch, err)
				}
				requantizedBuffer := requantizedValues[ch][:]
				if err := maindata.Requantize(h.GetSampleRate(), gc, &scalefactors, spectralBuffer, &requantizedBuffer); err != nil {
					t.Fatalf("file=%s frame=%d gr=%d ch=%d: failed to requantize values: %v", fileLabel, frameIndex, gr, ch, err)
				}
				reorderedBuffer := reorderedValues[ch][:]
				if err := maindata.Reorder(h.GetSampleRate(), gc, requantizedBuffer, &reorderedBuffer); err != nil {
					t.Fatalf("file=%s frame=%d gr=%d ch=%d: failed to reorder values: %v", fileLabel, frameIndex, gr, ch, err)
				}
				nonZeroRequantized := 0
				for _, v := range requantizedBuffer {
					if v != 0 {
						nonZeroRequantized++
					}
				}
				nonZeroReordered := 0
				for _, v := range reorderedBuffer {
					if v != 0 {
						nonZeroReordered++
					}
				}
				frameSummary = append(frameSummary, fmt.Sprintf(
					"gr=%d ch=%d part23=%d part2=%d part3=%d bigValues=%d bigValueLines=%d count1Lines=%d globalGain=%d scalefacCompress=%d tableSelect=%v subblockGain=%v region0=%d region1=%d windowSwitching=%v blockType=%s mixed=%v preflag=%v scalefacScale=%v count1Table=%v long=%v short=%v spectralLines=%d requantizedNonZero=%d reorderedNonZero=%d",
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
					nonZeroReordered,
				))

				br.Pos = part23End
				if br.Pos > len(mainData)*8 {
					t.Fatalf("file=%s frame=%d gr=%d ch=%d: part23 overruns main data bitstream", fileLabel, frameIndex, gr, ch)
				}
			}
		}

		t.Logf("file=%s frame=%d %s", fileLabel, frameIndex, strings.Join(frameSummary, " | "))
		frameIndex++
	}

	if frameIndex == 0 {
		t.Fatalf("no frames parsed from %s", fileLabel)
	}
}
