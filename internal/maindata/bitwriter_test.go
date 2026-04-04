package maindata

// bitWriter is a test helper that writes bits to a byte slice

type bitWriter struct {
	b    []byte
	cur  byte
	nbit int
}

func (w *bitWriter) write(n int, v uint32) {
	for i := n - 1; i >= 0; i-- {
		bit := byte((v >> uint(i)) & 1)
		w.cur = (w.cur << 1) | bit
		w.nbit++
		if w.nbit == 8 {
			w.b = append(w.b, w.cur)
			w.cur = 0
			w.nbit = 0
		}
	}
}

func (w *bitWriter) bytes() []byte {
	if w.nbit != 0 {
		for w.nbit != 0 {
			w.cur <<= 1
			w.nbit++
			if w.nbit == 8 {
				w.b = append(w.b, w.cur)
				w.cur = 0
				w.nbit = 0
			}
		}
	}
	return w.b
}
