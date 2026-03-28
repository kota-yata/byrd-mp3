package byrd

import (
	"fmt"
	"io"
)

type BitReader struct {
	data []byte
	pos  int // bit position
}

func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data}
}

// read arbitral amount of bits (up to 32) and return as uint32
func (r *BitReader) ReadBits(n int) (uint32, error) {
	if n <= 0 || n > 32 {
		return 0, fmt.Errorf("invalid bit count: %d", n)
	}
	if r.pos+n > len(r.data)*8 {
		return 0, io.ErrUnexpectedEOF
	}

	var v uint32
	for i := 0; i < n; i++ {
		byteIdx := (r.pos + i) / 8
		bitIdx := 7 - ((r.pos + i) % 8)
		bit := (r.data[byteIdx] >> bitIdx) & 0b1
		v = (v << 1) | uint32(bit)
	}
	r.pos += n
	return v, nil
}
