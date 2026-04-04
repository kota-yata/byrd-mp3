package stereo

import (
	"byrd/internal/header"
	"fmt"
	"math"
)

const MS_STEREO_SCALE = 0.7071067811865476 // 1 / sqrt(2)

func ApplyMSStereo(left []float64, right []float64) error {
	if len(left) != 576 || len(right) != 576 {
		return fmt.Errorf("ms stereo requires 576 spectral lines: left=%d right=%d", len(left), len(right))
	}

	for i := range left {
		m := left[i]
		s := right[i]
		left[i] = (m + s) * MS_STEREO_SCALE
		right[i] = (m - s) * MS_STEREO_SCALE
	}

	return nil
}

func ApplyJointStereo(channelMode header.ChannelMode, modeExt header.ModeExtension, left []float64, right []float64) error {
	if channelMode != header.ChannelModeJointStereo {
		return nil
	}
	if modeExt != header.ModeExtensionMSStereo && modeExt != header.ModeExtensionIntensityAndMS {
		return nil
	}
	if err := ApplyMSStereo(left, right); err != nil {
		return err
	}

	// TODO: implement intensity stereo when modeExt is IntensityStereo or IntensityAndMS.
	return nil
}

func almostEqual(a, b float64) bool {
	const epsilon = 1e-12
	return math.Abs(a-b) <= epsilon
}
