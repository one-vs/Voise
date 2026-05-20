package audio

import (
	"encoding/binary"
	"github.com/zaf/g711"
)

// MulawToPCM16 converts μ-law bytes to PCM 16-bit.
func MulawToPCM16(mulaw []byte) []int16 {
	pcmBytes := g711.DecodeUlaw(mulaw)
	return DecodePCM16LE(pcmBytes)
}

// PCM16ToMulaw converts PCM 16-bit to μ-law bytes.
func PCM16ToMulaw(pcm []int16) []byte {
	pcmBytes := EncodePCM16LE(pcm)
	return g711.EncodeUlaw(pcmBytes)
}

// DecodePCM16LE converts little-endian byte slice to int16 slice.
func DecodePCM16LE(b []byte) []int16 {
	pcm := make([]int16, len(b)/2)
	for i := 0; i < len(pcm); i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return pcm
}

// EncodePCM16LE converts int16 slice to little-endian byte slice.
func EncodePCM16LE(pcm []int16) []byte {
	b := make([]byte, len(pcm)*2)
	for i, v := range pcm {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(v))
	}
	return b
}
