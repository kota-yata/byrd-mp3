package byrd

import (
	"encoding/binary"
	"fmt"
	"os"
)

func WriteWAVFile(path string, pcm *PCMData) error {
	if pcm == nil {
		return fmt.Errorf("pcm data is nil")
	}
	if pcm.SampleRate == 0 {
		return fmt.Errorf("invalid sample rate: 0")
	}
	if pcm.Channels <= 0 {
		return fmt.Errorf("invalid channel count: %d", pcm.Channels)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	const bitsPerSample = 16
	const bytesPerSample = bitsPerSample / 8

	blockAlign := uint16(pcm.Channels * bytesPerSample)
	byteRate := uint32(pcm.SampleRate) * uint32(blockAlign)
	dataSize := uint32(len(pcm.Samples) * bytesPerSample)
	riffSize := 36 + dataSize

	if _, err := f.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, riffSize); err != nil {
		return err
	}
	if _, err := f.Write([]byte("WAVE")); err != nil {
		return err
	}

	if _, err := f.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(pcm.Channels)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(pcm.SampleRate)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(bitsPerSample)); err != nil {
		return err
	}

	if _, err := f.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, dataSize); err != nil {
		return err
	}
	for _, sample := range pcm.Samples {
		if err := binary.Write(f, binary.LittleEndian, sample); err != nil {
			return err
		}
	}

	return nil
}
