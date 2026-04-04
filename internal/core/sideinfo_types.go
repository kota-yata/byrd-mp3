package core

type SideInfo struct {
	MainDataBegin uint16
	SCFSI         [2][4]byte
	Granule       [2][2]GranuleChannelInfo
}

type GranuleChannelInfo struct {
	Part23Length     uint16
	BigValues        uint16
	GlobalGain       byte
	ScalefacCompress byte

	TableSelect  [3]byte
	SubblockGain [3]byte
	Region0Count byte
	Region1Count byte
	flags        byte
}

const PURE_SHORT_REGION0_COUNT = 8
const PURE_SHORT_REGION1_COUNT = 12
const MIXED_BLOCK_REGION0_COUNT = 7
const MIXED_BLOCK_REGION1_COUNT = 13

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
