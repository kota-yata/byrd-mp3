package byrd

import (
	"bufio"
	"fmt"
	"io"
)

type SideInfo struct {
	MainDataBegin uint16     // start of the main data to jump back to the previous frames if not 0
	SCFSI         [2][4]byte // bits to indicate if scalefactors are reused from previous granule for each band in each channel
	Granule       [2][2]GranuleChannelInfo
}

type GranuleChannelInfo struct {
	Part23Length     uint16 // total bit length of scalefactors and huffman-coded spectral information
	BigValues        uint16
	GlobalGain       byte // used for dequantization
	ScalefacCompress byte // bit length of each scalefactor value

	TableSelect  [3]byte
	SubblockGain [3]byte
	Region0Count byte
	Region1Count byte
	// WindowSwitching (1), BlockType(2), MixedBlockFlag(1), Preflag(1), ScalefacScale(1), Count1TableSelect(1), unused(1)
	flags byte
	// WindowSwitching indicates whether the block is short or long
	// BlockType indicates the block type.
	// MixedBlockFlag indicates whether the block is mixed (only valid if BlockTypeShort).
}

func GetSideInfoLength(h *MP3FrameHeader) int {
	if h.GetChannelMode() == ChannelModeMono {
		return 17
	}
	return 32
}

func ReadSideInfo(h *MP3FrameHeader, r *bufio.Reader, n int) (*SideInfo, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	if h.HasCRC() {
		h.crcTarget = append(h.crcTarget, buf...)
	}

	br := NewBitReader(buf)
	si := &SideInfo{}

	v, err := br.ReadBits(9)
	if err != nil {
		return nil, err
	}
	si.MainDataBegin = uint16(v)

	channels := 2
	if h.GetChannelMode() == ChannelModeMono {
		channels = 1
		_, err = br.ReadBits(5) // read and ignore private_bits
	} else {
		_, err = br.ReadBits(3) // read and ignore private_bits
	}
	if err != nil {
		return nil, err
	}

	for ch := range channels {
		for band := range 4 {
			v, err = br.ReadBits(1)
			if err != nil {
				return nil, err
			}
			si.SCFSI[ch][band] = byte(v)
		}
	}

	for gr := range 2 {
		for ch := range channels {
			gc := &si.Granule[gr][ch]

			v, err = br.ReadBits(12)
			if err != nil {
				return nil, err
			}
			gc.Part23Length = uint16(v)

			v, err = br.ReadBits(9)
			if err != nil {
				return nil, err
			}
			gc.BigValues = uint16(v)

			v, err = br.ReadBits(8)
			if err != nil {
				return nil, err
			}
			gc.GlobalGain = byte(v)

			v, err = br.ReadBits(4)
			if err != nil {
				return nil, err
			}
			gc.ScalefacCompress = byte(v)

			v, err = br.ReadBits(1)
			if err != nil {
				return nil, err
			}
			gc.SetWindowSwitching(v == 1)

			if gc.GetWindowSwitching() {
				v, err = br.ReadBits(2)
				if err != nil {
					return nil, err
				}
				gc.SetBlockType(BlockType(v))

				v, err = br.ReadBits(1)
				if err != nil {
					return nil, err
				}
				gc.SetMixedBlockFlag(v == 1)

				for i := 0; i < 2; i++ {
					v, err = br.ReadBits(5)
					if err != nil {
						return nil, err
					}
					gc.TableSelect[i] = byte(v)
				}
				gc.TableSelect[2] = 0

				for i := 0; i < 3; i++ {
					v, err = br.ReadBits(3)
					if err != nil {
						return nil, err
					}
					gc.SubblockGain[i] = byte(v)
				}

				// window_switching==1 means block is either short or mixed, so long block here is invalid
				if gc.GetBlockType() == BlockTypeLong {
					return nil, fmt.Errorf("invalid side info: block_type=0 with window_switching=1")
				}

				if gc.GetBlockType() == BlockTypeShort && !gc.GetMixedBlockFlag() {
					gc.Region0Count = 8
				} else {
					gc.Region0Count = 7
				}
				gc.Region1Count = 20 - gc.Region0Count
			} else {
				gc.SetBlockType(BlockTypeLong)
				gc.SetMixedBlockFlag(false)
				gc.SubblockGain = [3]byte{}

				for i := 0; i < 3; i++ {
					v, err = br.ReadBits(5)
					if err != nil {
						return nil, err
					}
					gc.TableSelect[i] = byte(v)
				}

				v, err = br.ReadBits(4)
				if err != nil {
					return nil, err
				}
				gc.Region0Count = byte(v)

				v, err = br.ReadBits(3)
				if err != nil {
					return nil, err
				}
				gc.Region1Count = byte(v)
			}

			v, err = br.ReadBits(1)
			if err != nil {
				return nil, err
			}
			gc.SetPreflag(v == 1)

			v, err = br.ReadBits(1)
			if err != nil {
				return nil, err
			}
			gc.SetScalefacScale(v == 1)

			v, err = br.ReadBits(1)
			if err != nil {
				return nil, err
			}
			gc.SetCount1TableSelect(v == 1)
		}
	}

	return si, nil
}

// getter/setter functions for the flag

func (g *GranuleChannelInfo) SetWindowSwitching(v bool) {
	g.flags &^= 1 << 7
	if v {
		g.flags |= 1 << 7
	}
}
func (g *GranuleChannelInfo) GetWindowSwitching() bool {
	return (g.flags>>7)&1 == 1
}

func (g *GranuleChannelInfo) SetBlockType(v BlockType) {
	g.flags &^= 0b11 << 5
	g.flags |= (byte(v) & 0b11) << 5
}
func (g *GranuleChannelInfo) GetBlockType() BlockType {
	return BlockType((g.flags >> 5) & 0b11)
}

func (g *GranuleChannelInfo) SetMixedBlockFlag(v bool) {
	g.flags &^= 1 << 4
	if v {
		g.flags |= 1 << 4
	}
}
func (g *GranuleChannelInfo) GetMixedBlockFlag() bool {
	return (g.flags>>4)&1 == 1
}

func (g *GranuleChannelInfo) SetPreflag(v bool) {
	g.flags &^= 1 << 3
	if v {
		g.flags |= 1 << 3
	}
}
func (g *GranuleChannelInfo) GetPreflag() bool {
	return (g.flags>>3)&1 == 1
}

func (g *GranuleChannelInfo) SetScalefacScale(v bool) {
	g.flags &^= 1 << 2
	if v {
		g.flags |= 1 << 2
	}
}
func (g *GranuleChannelInfo) GetScalefacScale() bool {
	return (g.flags>>2)&1 == 1
}

func (g *GranuleChannelInfo) SetCount1TableSelect(v bool) {
	g.flags &^= 1 << 1
	if v {
		g.flags |= 1 << 1
	}
}
func (g *GranuleChannelInfo) GetCount1TableSelect() bool {
	return (g.flags>>1)&1 == 1
}
