package common

type ChannelMode uint8

const (
	ChannelModeStereo      ChannelMode = iota // 0b00
	ChannelModeJointStereo                    // 0b01
	ChannelModeDualChannel                    // 0b10
	ChannelModeMono                           // 0b11
)

func (m ChannelMode) String() string {
	switch m {
	case ChannelModeStereo:
		return "Stereo"
	case ChannelModeJointStereo:
		return "JointStereo"
	case ChannelModeDualChannel:
		return "DualChannel"
	case ChannelModeMono:
		return "Mono"
	default:
		return "Unknown"
	}
}

type ModeExtension uint8

const (
	ModeExtensionNone             ModeExtension = 0b00
	ModeExtensionIntensityStereo  ModeExtension = 0b01
	ModeExtensionMSStereo         ModeExtension = 0b10
	ModeExtensionIntensityAndMS   ModeExtension = 0b11
)

func (m ModeExtension) String() string {
	switch m {
	case ModeExtensionNone:
		return "None"
	case ModeExtensionIntensityStereo:
		return "IntensityStereo"
	case ModeExtensionMSStereo:
		return "MSStereo"
	case ModeExtensionIntensityAndMS:
		return "IntensityAndMS"
	default:
		return "Unknown"
	}
}

type BlockType uint8

const (
	BlockTypeLong BlockType = iota
	BlockTypeStart
	BlockTypeShort
	BlockTypeEnd
)

func (b BlockType) String() string {
	switch b {
	case BlockTypeLong:
		return "Long"
	case BlockTypeStart:
		return "Start"
	case BlockTypeShort:
		return "Short"
	case BlockTypeEnd:
		return "End"
	default:
		return "Unknown"
	}
}

type ScalefactorBits struct {
	Slen1 int
	Slen2 int
}
