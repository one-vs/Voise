package audio

import (
	"sync"
)

type JitterBuffer struct {
	data []int16
	mu   sync.Mutex
}

func NewJitterBuffer() *JitterBuffer {
	return &JitterBuffer{
		data: make([]int16, 0, 48000), // 1 second of 48kHz audio buffer
	}
}

func (jb *JitterBuffer) Push(samples []int16) {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	jb.data = append(jb.data, samples...)
}

func (jb *JitterBuffer) Pop(n int) []int16 {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.data) == 0 {
		return make([]int16, n)
	}

	if len(jb.data) < n {
		res := make([]int16, n)
		copy(res, jb.data)
		jb.data = jb.data[:0]
		return res
	}

	res := jb.data[:n]
	jb.data = jb.data[n:]
	// Need to copy to prevent memory leak if jb.data grows large then shrinks
	resCopy := make([]int16, n)
	copy(resCopy, res)
	return resCopy
}

func (jb *JitterBuffer) Clear() {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	jb.data = jb.data[:0]
}
