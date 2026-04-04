package synthesis

// Quantize to int16 PCM sample

import "math"

func QuantizeSample(sample float64) int16 {
	s := int(math.Round(sample * 32767)) // 16-bit PCM
	if s > 32767 {
		s = 32767
	}
	if s < -32768 {
		s = -32768
	}
	return int16(s)
}
