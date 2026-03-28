package byrd

// MPEG-1 Layer III frame parser

import (
	"bufio"
	"fmt"
	"io"
)

type MP3FrameHeader struct {
	// version (1), protection (1), bitrate index(4), sample rate index(2),
	// version=0 indicates MPEG2, version=1 indicates MPEG1
	flag1 byte
	// channel mode (2), mode extension (2), copyright (1), original (1), emphasis (2)
	flag2   byte
	padding bool
	// CRC16 value read from the frame (if present)
	crcValue [2]byte
	// Bytes covered by MPEG audio CRC: header bytes 3-4 and the whole Layer III side info
	crcTarget []byte
}

func ReadHeader(h *MP3FrameHeader, reader *bufio.Reader) error {
	var b byte
	var err error
	*h = MP3FrameHeader{} // reset frame state
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
		if b&0b11100000 != 0b11100000 {
			// back off one byte because it might be the start of the next frame or an ID3 tag
			if err := reader.UnreadByte(); err != nil {
				return err
			}
			continue
		}
		// version check
		switch (b >> 3) & 0b11 {
		case 0b11: // MPEG Version 1.0
			h.flag1 |= 1 << 7 // set msb of flag1 to indicate MPEG1
		default:
			// ignore MPEG Version 2/2.5 at this point
			return fmt.Errorf("unsupported MPEG version %02x", (b>>3)&0b11)
		}
		// at this time we only support Layer III
		if (b >> 1 & 0b11) != 0b01 {
			return fmt.Errorf("unsupported layer, only Layer III (MP3) is supported")
		}
		fmt.Printf("Found potential MP3 frame header: %02x %02x\n", 0xFF, b)
		// read protection bit, 0 means CRC is present, 1 means no CRC
		pBit := b & 0b01
		hasCRC := pBit == 0
		fmt.Printf("Protection bit: %d (has CRC: %v)\n", pBit, hasCRC)
		h.flag1 |= pBit << 6 // set protection bit

		b, err = reader.ReadByte()
		if err != nil {
			return err
		}
		if hasCRC {
			h.crcTarget = make([]byte, 2)
			h.crcTarget[0] = b
		}
		fmt.Printf("Bitrate index: %d\n", (b>>4)&0b1111)
		bitrateIndex := (b >> 4) & 0b1111
		if bitrateIndex == 0b1111 {
			return fmt.Errorf("invalid bitrate index: bad")
		}
		h.flag1 |= bitrateIndex << 2 // set bitrate index

		sampleRateIndex := (b >> 2) & 0b11
		if sampleRateIndex == 0b11 {
			return fmt.Errorf("invalid sample rate index: reserved")
		}
		h.flag1 |= sampleRateIndex // set sample rate index

		h.padding = ((b >> 1) & 0b01) == 1 // set padding

		// the remaining one bit is private bit, we ignore it

		h.flag2, err = reader.ReadByte()
		if err != nil {
			return err
		}
		fmt.Printf("Channel mode: %s\n", h.GetChannelMode())
		if hasCRC {
			h.crcTarget[1] = h.flag2
		}

		// if CRC is present, read 2 bytes after the header which is CRC value calculated on the sender side
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
// the magic number 144 is derived from 1152 samples per frame and 8 bits
func (h *MP3FrameHeader) GetFrameLength() (int, error) {
	bitrateKbps, isFree := h.GetBitrateKbps()
	if isFree {
		return 0, fmt.Errorf("free bitrate frames are not supported")
	}
	sampleRate := h.GetSampleRate()
	pad := 0
	if h.Padding() {
		pad = 1
	}

	frameLength := (144 * int(bitrateKbps*1000) / int(sampleRate)) + pad
	return frameLength, nil
}

func (h *MP3FrameHeader) ValidateCRC(reader *bufio.Reader) bool {
	if !h.HasCRC() {
		return true // no CRC to validate
	}
	return true // TODO: implement CRC16 validation
}

// GetBitrate returns the bitrate in bps. If the frame uses free bitrate, returns (0, true)
func (h *MP3FrameHeader) GetBitrateKbps() (uint16, bool) {
	bitrateIndex := (h.flag1 >> 2) & 0b1111
	if bitrateIndex == 0 {
		return 0, true // free bitrate
	}
	return V1L3_BITRATE_TABLE[bitrateIndex], false
}

func (h *MP3FrameHeader) GetSampleRate() uint16 {
	sampleRateIndex := h.flag1 & 0b11
	return V1_SAMPLE_RATE_TABLE[sampleRateIndex]
}

// getter functions

func (h *MP3FrameHeader) HasCRC() bool {
	return (h.flag1 & (1 << 6)) == 0
}

func (h *MP3FrameHeader) Padding() bool {
	return h.padding
}

func (h *MP3FrameHeader) GetChannelMode() ChannelMode {
	return ChannelMode((h.flag2 >> 6) & 0b11)
}

func (h *MP3FrameHeader) GetModeExtension() byte {
	if h.GetChannelMode() != ChannelModeJointStereo {
		return 0
	}
	return (h.flag2 >> 4) & 0b11
}

func (h *MP3FrameHeader) IsCopyrighted() bool {
	return ((h.flag2 >> 3) & 0b1) == 1
}

func (h *MP3FrameHeader) IsOriginal() bool {
	return ((h.flag2 >> 2) & 0b1) == 1
}

func (h *MP3FrameHeader) GetEmphasis() byte {
	return h.flag2 & 0b11
}
