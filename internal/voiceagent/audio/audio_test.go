package audio

import (
	"testing"
)

func TestMulawPCM16Roundtrip(t *testing.T) {
	// A few sample values
	originalPCM := []int16{0, 100, -100, 1000, -1000, 30000, -30000}

	mulaw := PCM16ToMulaw(originalPCM)
	backToPCM := MulawToPCM16(mulaw)

	if len(originalPCM) != len(backToPCM) {
		t.Errorf("Length mismatch: %d != %d", len(originalPCM), len(backToPCM))
	}

	for i := range originalPCM {
		// μ-law is lossy, so we don't expect exact same values,
		// but they should be reasonably close.
		diff := originalPCM[i] - backToPCM[i]
		if diff < 0 {
			diff = -diff
		}

		// For μ-law, relative error is roughly constant (around 3-5%).
		// We allow some margin.
		limit := int16(float64(originalPCM[i])*0.1)
		if limit < 0 { limit = -limit }
		if limit < 50 { limit = 50 } // Minimum absolute error for small values

		if diff > limit {
			t.Errorf("Value at %d differs too much: %d -> %d (diff %d, limit %d)", i, originalPCM[i], backToPCM[i], diff, limit)
		}
	}
}

func TestResample(t *testing.T) {
	in := []int16{0, 1000, 2000, 1000, 0, -1000, -2000, -1000}

	// Upsample 1:2
	out := Resample(in, 8000, 16000)
	if len(out) != len(in)*2 {
		t.Errorf("Expected length %d, got %d", len(in)*2, len(out))
	}

	// Check a few interpolated values
	// in[0]=0, in[1]=1000. Midpoint should be ~500.
	if out[1] < 450 || out[1] > 550 {
		t.Errorf("Expected interpolated value ~500, got %d", out[1])
	}

	// Downsample 2:1
	in2 := make([]int16, 16)
	for i := range in2 {
		in2[i] = int16(i * 100)
	}
	out2 := Resample(in2, 16000, 8000)
	if len(out2) != len(in2)/2 {
		t.Errorf("Expected length %d, got %d", len(in2)/2, len(out2))
	}
}
