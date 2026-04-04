package byrd

import (
	"bufio"
	"byrd/internal/decoder"
)

type PCMData struct {
	Samples    []int16
	SampleRate uint16
	Channels   int
}

func DecodeMP3File(path string) (*PCMData, error) {
	f, err := decoder.OpenMP3File(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	samples, sampleRate, channels, err := decoder.DecodeMP3Frames(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	return &PCMData{
		Samples:    samples,
		SampleRate: sampleRate,
		Channels:   channels,
	}, nil
}

func ConvertMP3FileToWAV(mp3Path, wavPath string) error {
	pcm, err := DecodeMP3File(mp3Path)
	if err != nil {
		return err
	}
	return WriteWAVFile(wavPath, pcm)
}
