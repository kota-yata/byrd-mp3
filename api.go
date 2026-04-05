package byrd

import (
	"bufio"
	"byrd/internal/decoder"
	"encoding/binary"
	"io"
)

type PCMData struct {
	Samples    []int16
	SampleRate uint16
	Channels   int
}

type Decoder struct {
	data       []byte
	pos        int64
	sampleRate int
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	pcm, err := decodeMP3(r)
	if err != nil {
		return nil, err
	}
	return &Decoder{
		data:       pcmToStereoBytes(pcm),
		sampleRate: int(pcm.SampleRate),
	}, nil
}

func (d *Decoder) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if d.pos >= int64(len(d.data)) {
		return 0, io.EOF
	}
	n := copy(p, d.data[d.pos:])
	d.pos += int64(n)
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = d.pos + offset
	case io.SeekEnd:
		next = int64(len(d.data)) + offset
	default:
		return 0, io.ErrUnexpectedEOF
	}
	if next < 0 {
		return 0, io.ErrUnexpectedEOF
	}
	d.pos = next
	return d.pos, nil
}

func (d *Decoder) SampleRate() int {
	return d.sampleRate
}

func (d *Decoder) Length() int64 {
	return int64(len(d.data))
}

func DecodeMP3File(path string) (*PCMData, error) {
	f, err := decoder.OpenMP3File(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return decodeMP3(f)
}

func ConvertMP3FileToWAV(mp3Path, wavPath string) error {
	pcm, err := DecodeMP3File(mp3Path)
	if err != nil {
		return err
	}
	return WriteWAVFile(wavPath, pcm)
}

func decodeMP3(r io.Reader) (*PCMData, error) {
	samples, sampleRate, channels, err := decoder.DecodeMP3Frames(bufio.NewReader(r))
	if err != nil {
		return nil, err
	}
	return &PCMData{
		Samples:    samples,
		SampleRate: sampleRate,
		Channels:   channels,
	}, nil
}

func pcmToStereoBytes(pcm *PCMData) []byte {
	if pcm == nil || len(pcm.Samples) == 0 {
		return nil
	}
	switch pcm.Channels {
	case 1:
		out := make([]byte, len(pcm.Samples)*4)
		for i, sample := range pcm.Samples {
			binary.LittleEndian.PutUint16(out[i*4:], uint16(sample))
			binary.LittleEndian.PutUint16(out[i*4+2:], uint16(sample))
		}
		return out
	default:
		out := make([]byte, len(pcm.Samples)*2)
		for i, sample := range pcm.Samples {
			binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
		}
		return out
	}
}
