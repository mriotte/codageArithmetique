package bits_io

import (
	"errors"
	"fmt"
)

type BitReader struct {
	bytes  []byte // bits that have been written; the last byte may be incomplet, with only the most significant bits being relevant
	offset int    // number of bits that have been read
	len    int    // total number of bits in buffer (constant)
}

func NewReader(bs []byte) BitReader {
	return BitReader{bs, 0, len(bs) * 8}
}

func (b BitReader) Len() int {
	return b.len - b.offset
}

func (b *BitReader) ReadBits(nb int) (uint64, error) {
	if nb > 64 {
		return 0, errors.New(fmt.Sprintf("BitReader.ReadBits: cannot read %d bits at once", nb))
	}
	if b.offset+nb > b.len {
		return 0, errors.New(fmt.Sprintf("BitReader.ReadBits: not enough bits in buffer", nb))
	}

	w := uint64(0)
	for nb > 0 {
		c := b.bytes[b.offset/8]
		s := 8 - b.offset%8
		n := c & (1<<s - 1)
		t := min(s, nb)
		n = n >> (s - t)
		w = w<<t | uint64(n)
		nb -= t
		b.offset += t
	}
	return w, nil
}

func (b *BitReader) Unpad() error {
	last := len(b.bytes) - 1
	for i := range 8 {
		c := b.bytes[last] & (1<<(i+1) - 1)
		if c == 1<<i {
			b.len -= i + 1
			if i+1 == 8 {
				b.bytes = b.bytes[0:last]
			}
			return nil
		}
	}
	// that could only happen if the last byte is 0, which can only happen if
	// we did not add padding
	return errors.New("BitWriter.Unpad: no padding found")
}
