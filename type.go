package byrd

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
