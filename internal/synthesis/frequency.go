package synthesis

func ApplyFrequencyInversion(samples *[32][18]float64) {
	if samples == nil {
		return
	}

	for sb := 1; sb < 32; sb += 2 {
		for i := 1; i < 18; i += 2 {
			samples[sb][i] = -samples[sb][i]
		}
	}
}
