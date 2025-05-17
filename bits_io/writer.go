package bits_io

import (
	"errors"
	"fmt"
)

type BitWriter struct {
	bytes []byte // bits that have been written; the last byte may be incomplet, with only the most significant bits being relevant
	len   int    // total number of bits in buffer
}

func NewWriter() BitWriter {
	var B BitWriter
	return B
}

func (b BitWriter) Len() int {
	return b.len
}

func (b *BitWriter) WriteBits(w uint64, nb int) error {
	if nb > 64 {
		return errors.New(fmt.Sprintf("BitWriter.WriteBits: cannot write %d bits at once", nb))
	}
	if nb < 63 && w >= 1<<nb {
		return errors.New(fmt.Sprintf("BitWriter.WriteBits: 0x%x doesn't fit on %d bits", w, nb))
	}

	for nb > 0 {
		s := b.len % 8
		if s == 0 {
			b.bytes = append(b.bytes, 0x00)
		}
		r := 8 - s                                      // nb of free bits in the last byte of B
		t := min(r, nb)                                 // nb of bits we're going to take from w
		msb := (w >> (nb - t))                          // the t most significant bits from nb
		b.bytes[len(b.bytes)-1] ^= byte(msb << (r - t)) // adding msb as far left as possible in the last byte of B
		w &= (1<<(nb-t) - 1)
		b.len += t
		nb -= t
	}
	return nil
}

func (b *BitWriter) Pad() {
	nb := 8 - (b.len)%8 // how many padding bits to add
	w := uint64(1) << (nb - 1)
	b.WriteBits(w, nb)
}

func (b *BitWriter) ToBytes() ([]byte, error) {
	if b.len%8 == 0 {
		return b.bytes[0 : b.len/8], nil
	}
	return nil, errors.New("BitWriter.ToBytes: length is not a multiple of 8")
}
