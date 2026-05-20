package audio

// Resample performs linear interpolation to resample PCM 16-bit audio from fromHz to toHz.
func Resample(in []int16, fromHz, toHz int) []int16 {
	if fromHz == toHz {
		return in
	}

	ratio := float64(fromHz) / float64(toHz)
	outLen := int(float64(len(in)) * float64(toHz) / float64(fromHz))
	out := make([]int16, outLen)

	for i := 0; i < outLen; i++ {
		pos := float64(i) * ratio
		idx := int(pos)
		frac := pos - float64(idx)

		if idx+1 < len(in) {
			out[i] = int16(float64(in[idx])*(1-frac) + float64(in[idx+1])*frac)
		} else {
			out[i] = in[idx]
		}
	}

	return out
}
