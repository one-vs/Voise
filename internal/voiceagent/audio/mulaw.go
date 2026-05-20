package audio

import (
	"encoding/binary"
	"github.com/zaf/g711"
)

// MulawToPCM16 converts μ-law bytes to PCM 16-bit.
func MulawToPCM16(mulaw []byte) []int16 {
	pcmBytes := g711.DecodeUlaw(mulaw)
	pcm := make([]int16, len(pcmBytes)/2)
	for i := 0; i < len(pcm); i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(pcmBytes[i*2:]))
	}
	return pcm
}

// PCM16ToMulaw converts PCM 16-bit to μ-law bytes.
func PCM16ToMulaw(pcm []int16) []byte {
	pcmBytes := make([]byte, len(pcm)*2)
	for i, v := range pcm {
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(v))
	}
	mulaw := g711.EncodeUlaw(pcmBytes)
	return mulaw
}
