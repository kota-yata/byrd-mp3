package core

import (
	"fmt"
	"io"
)

type BitReader struct {
	data []byte
	Pos  int // bit position
}

func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data}
}

// read arbitral amount of bits (up to 32) and return as uint32.
// caller should reuse the same variable for the retuned value to avoid unnecessary allocations
func (r *BitReader) ReadBits(n int) (uint32, error) {
	var v uint32
	if err := r.ReadBitsTo(&v, n); err != nil {
		return 0, err
	}
	return v, nil
}

// ReadBitsTo reads up to 32 bits into dst and advances the reader.
// Reusing dst lets callers avoid repeatedly materializing temporary values.
func (r *BitReader) ReadBitsTo(dst *uint32, n int) error {
	if dst == nil {
		return fmt.Errorf("nil destination")
	}
	if n <= 0 || n > 32 {
		return fmt.Errorf("invalid bit count: %d", n)
	}
	if r.Pos+n > len(r.data)*8 {
		return io.ErrUnexpectedEOF
	}

	var v uint32
	for i := 0; i < n; i++ {
		byteIdx := (r.Pos + i) / 8
		bitIdx := 7 - ((r.Pos + i) % 8)
		bit := (r.data[byteIdx] >> bitIdx) & 0b1
		v = (v << 1) | uint32(bit)
	}
	r.Pos += n
	*dst = v
	return nil
}
