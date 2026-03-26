package byrd

// MPEG-1 Layer III frame parser

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

type MP3Frame struct {
	// version (1), protection (1), bitrate index(4), sample rate index(2),
	// version=0 indicates MPEG2, version=1 indicates MPEG1
	flag1 byte
	// unused(5), padding (1), channel mode (2)
	flag2 byte
	// CRC16 value read from the frame (if present)
	crcValue [2]byte
	// Bytes covered by MPEG audio CRC: header bytes 3-4 and Layer III side info
	crcTarget []byte
}

func OpenMP3File(path string) (io.ReadCloser, error) {
	ext := ".mp3"
	if len(path) < len(ext) || path[len(path)-len(ext):] != ext {
		return nil, fmt.Errorf("unsupported file format: %s", path)
	}
	return os.Open(path)
}

func ReadHeader(h *MP3Frame, reader *bufio.Reader) error {
	var b byte
	var err error
	*h = MP3Frame{} // reset frame state
	for {
		b, err = reader.ReadByte()
		if err != nil {
			return err
		}
		if b != 0xFF {
			// possibly ID3 tag, keep searching
			continue
		}

		b, err = reader.ReadByte()
		if err != nil {
			return err
		}
		// sync bit is 11 bits, check the last 3 bits to confirm
		if b&0xE0 != 0xE0 {
			// back off one byte because it might be the start of the next frame or an ID3 tag
			if err := reader.UnreadByte(); err != nil {
				return err
			}
			continue
		}
		// version check
		switch (b >> 3) & 0x03 {
		case 0b11: // MPEG Version 1.0
			h.flag1 |= 1 << 7 // set msb of flag1 to indicate MPEG1
		default:
			// ignore MPEG Version 2/2.5 at this point
			return fmt.Errorf("unsupported MPEG version %02x", (b>>3)&0x03)
		}
		// at this time we only support Layer III
		if (b >> 1 & 0x03) != 0x01 {
			return fmt.Errorf("unsupported layer, only Layer III (MP3) is supported")
		}
		log.Printf("Found potential MP3 frame header: %02x %02x", 0xFF, b)
		// read protection bit, 0 means CRC is present, 1 means no CRC
		pBit := b & 0x01
		hasCRC := pBit == 0
		log.Printf("Protection bit: %d (has CRC: %v)", pBit, hasCRC)
		h.flag1 |= pBit << 6 // set protection bit

		b, err = reader.ReadByte()
		if err != nil {
			return err
		}
		if hasCRC {
			h.crcTarget = make([]byte, 2)
			h.crcTarget[0] = b
		}
		log.Printf("Bitrate index: %d", (b>>4)&0x0F)
		bitrateIndex := (b >> 4) & 0x0F
		if bitrateIndex == 0x0F {
			return fmt.Errorf("invalid bitrate index: bad")
		}
		h.flag1 |= bitrateIndex << 2 // set bitrate index

		sampleRateIndex := (b >> 2) & 0x03
		if sampleRateIndex == 0x03 {
			return fmt.Errorf("invalid sample rate index: reserved")
		}
		h.flag1 |= sampleRateIndex // set sample rate index

		h.flag2 |= ((b >> 1) & 0x01) << 1 // set padding bit

		// the remaining one bit is private bit, we ignore it

		b, err = reader.ReadByte()
		if err != nil {
			return err
		}
		log.Printf("Sample rate index: %d", (b>>2)&0x03)
		if hasCRC {
			h.crcTarget[1] = b
		}
		// set channel mode
		// 00 = Stereo, 01 = Joint stereo (Stereo), 10 = Dual channel (Stereo), 11 = Single channel (Mono)
		h.flag2 |= (b >> 6) & 0x03
		log.Printf("Channel mode: %02b", (b>>6)&0x03)

		// TODO: Mode extension, copyright, original, emphasis bits are ignored for now

		if hasCRC {
			_, err := io.ReadFull(reader, h.crcValue[:])
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// TODO: read payload

// Layer III frame length = (144 * Bitrate / SampleRate) + Padding
func GetFrameLength(h *MP3Frame) (int, error) {
	bitrateKbps, isFree := GetBitrateKbps(h)
	if isFree {
		return 0, fmt.Errorf("free bitrate frames are not supported")
	}
	sampleRate := GetSampleRate(h)
	padding := GetPadding(h)

	frameLength := (144 * int(bitrateKbps*1000) / int(sampleRate)) + int(padding)
	return frameLength, nil
}

func ValidateCRC(h *MP3Frame, reader *bufio.Reader) bool {
	if !hasCRC(h) {
		return true // no CRC to validate
	}
	return true // TODO: implement CRC16 validation
}

func hasCRC(h *MP3Frame) bool {
	return (h.flag1 & (1 << 6)) == 0
}

// IsStereo does not distinguish between joint stereo and dual channel modes
func isStereo(h *MP3Frame) bool {
	return (h.flag2 & 0x03) != 0x03
}

// GetBitrate returns the bitrate in bps. If the frame uses free bitrate, returns (0, true)
func GetBitrateKbps(h *MP3Frame) (uint16, bool) {
	bitrateIndex := (h.flag1 >> 2) & 0x0F
	if bitrateIndex == 0 {
		return 0, true // free bitrate
	}
	version := (h.flag1 >> 7) & 0x01
	if version == 1 {
		return V1L3_BITRATE_TABLE[bitrateIndex], false
	} else {
		return V2L3_BITRATE_TABLE[bitrateIndex], false
	}
}

func GetSampleRate(h *MP3Frame) uint16 {
	sampleRateIndex := h.flag1 & 0x03
	version := (h.flag1 >> 7) & 0x01
	if version == 1 {
		return V1_SAMPLE_RATE_TABLE[sampleRateIndex]
	} else {
		return V2_SAMPLE_RATE_TABLE[sampleRateIndex]
	}
}

func GetPadding(h *MP3Frame) uint8 {
	return (h.flag2 >> 1) & 0x01
}
