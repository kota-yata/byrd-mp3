package stereo

import (
	"byrd/internal/header"
	"math"
	"testing"
)

func TestApplyMSStereo(t *testing.T) {
	left := make([]float64, 576)
	right := make([]float64, 576)
	left[0] = 2
	right[0] = 0
	left[1] = 3
	right[1] = 1

	if err := ApplyMSStereo(left, right); err != nil {
		t.Fatalf("ApplyMSStereo failed: %v", err)
	}

	if !almostEqual(left[0], 2*MS_STEREO_SCALE) || !almostEqual(right[0], 2*MS_STEREO_SCALE) {
		t.Fatalf("line 0 got left=%f right=%f", left[0], right[0])
	}
	if !almostEqual(left[1], 4*MS_STEREO_SCALE) || !almostEqual(right[1], 2*MS_STEREO_SCALE) {
		t.Fatalf("line 1 got left=%f right=%f", left[1], right[1])
	}
}

func TestApplyJointStereo_ModeExtMSOnly(t *testing.T) {
	left := make([]float64, 576)
	right := make([]float64, 576)
	left[0] = 1
	right[0] = -1

	if err := ApplyJointStereo(header.ChannelModeJointStereo, header.ModeExtensionMSStereo, left, right); err != nil {
		t.Fatalf("ApplyJointStereo failed: %v", err)
	}

	if !almostEqual(left[0], 0) || !almostEqual(right[0], 2*MS_STEREO_SCALE) {
		t.Fatalf("got left=%f right=%f", left[0], right[0])
	}
}

func TestApplyJointStereo_NoOpWhenDisabled(t *testing.T) {
	left := make([]float64, 576)
	right := make([]float64, 576)
	left[0] = 5
	right[0] = 2

	if err := ApplyJointStereo(header.ChannelModeStereo, header.ModeExtensionMSStereo, left, right); err != nil {
		t.Fatalf("ApplyJointStereo failed: %v", err)
	}
	if left[0] != 5 || right[0] != 2 {
		t.Fatalf("non-joint stereo should be unchanged, got left=%f right=%f", left[0], right[0])
	}

	if err := ApplyJointStereo(header.ChannelModeJointStereo, header.ModeExtensionIntensityStereo, left, right); err != nil {
		t.Fatalf("ApplyJointStereo failed: %v", err)
	}
	if left[0] != 5 || right[0] != 2 {
		t.Fatalf("joint stereo without ms flag should be unchanged, got left=%f right=%f", left[0], right[0])
	}
}

func TestApplyMSStereo_InvalidLength(t *testing.T) {
	if err := ApplyMSStereo(make([]float64, 10), make([]float64, 576)); err == nil {
		t.Fatalf("expected invalid length error")
	}
}

func almostEqual(a, b float64) bool {
	const epsilon = 1e-12
	return math.Abs(a-b) <= epsilon
}
