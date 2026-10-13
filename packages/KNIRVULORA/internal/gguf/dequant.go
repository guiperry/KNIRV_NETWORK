package gguf

import (
	"encoding/binary"
	"fmt"
	"math"
)

func f16(b []byte) float32 {
	h := binary.LittleEndian.Uint16(b)
	sign := uint32(h>>15) << 31
	exp := int32(h>>10) & 0x1f
	frac := uint32(h) & 0x3ff
	switch {
	case exp == 0 && frac == 0:
		return math.Float32frombits(sign)
	case exp == 0: // subnormal
		for frac&0x400 == 0 {
			frac <<= 1
			exp--
		}
		exp++
		frac &= 0x3ff
		return math.Float32frombits(sign | uint32(exp+127-15)<<23 | frac<<13)
	case exp == 0x1f:
		return math.Float32frombits(sign | 0x7f800000 | frac<<13)
	}
	return math.Float32frombits(sign | uint32(exp+127-15)<<23 | frac<<13)
}

// block describes a block-quantised layout: elements per block, bytes per block.
func block(t TensorType) (elems, bytes int, ok bool) {
	switch t {
	case Q4_0:
		return 32, 18, true
	case Q4_1:
		return 32, 20, true
	case Q5_0:
		return 32, 22, true
	case Q5_1:
		return 32, 24, true
	case Q8_0:
		return 32, 34, true
	}
	return 0, 0, false
}

func byteSize(t TensorType, n int) (int, error) {
	switch t {
	case F32:
		return n * 4, nil
	case F16, BF16:
		return n * 2, nil
	}
	if e, b, ok := block(t); ok {
		if n%e != 0 {
			return 0, fmt.Errorf("%d elements is not a multiple of the %s block size %d", n, t, e)
		}
		return n / e * b, nil
	}
	return 0, fmt.Errorf("%s weights are not supported for connector derivation (supported: F32, F16, BF16, Q4_0, Q4_1, Q5_0, Q5_1, Q8_0); use a model in one of those formats or a higher-precision copy", t)
}

func dequantize(t TensorType, raw []byte, n int) ([]float32, error) {
	out := make([]float32, n)
	switch t {
	case F32:
		for i := range out {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
		return out, nil
	case F16:
		for i := range out {
			out[i] = f16(raw[i*2:])
		}
		return out, nil
	case BF16:
		for i := range out {
			out[i] = math.Float32frombits(uint32(binary.LittleEndian.Uint16(raw[i*2:])) << 16)
		}
		return out, nil
	}
	e, b, ok := block(t)
	if !ok {
		return nil, fmt.Errorf("unsupported type %s", t)
	}
	for blk := 0; blk < n/e; blk++ {
		src, dst := raw[blk*b:(blk+1)*b], out[blk*e:(blk+1)*e]
		switch t {
		case Q4_0:
			d := f16(src)
			for j := 0; j < 16; j++ {
				q := src[2+j]
				dst[j] = float32(int(q&0xf)-8) * d
				dst[j+16] = float32(int(q>>4)-8) * d
			}
		case Q4_1:
			d, m := f16(src), f16(src[2:])
			for j := 0; j < 16; j++ {
				q := src[4+j]
				dst[j] = float32(q&0xf)*d + m
				dst[j+16] = float32(q>>4)*d + m
			}
		case Q5_0:
			d := f16(src)
			qh := binary.LittleEndian.Uint32(src[2:])
			for j := 0; j < 16; j++ {
				q := src[6+j]
				x0 := int(q&0xf) | int((qh>>uint(j))&1)<<4
				x1 := int(q>>4) | int((qh>>uint(j+16))&1)<<4
				dst[j] = float32(x0-16) * d
				dst[j+16] = float32(x1-16) * d
			}
		case Q5_1:
			d, m := f16(src), f16(src[2:])
			qh := binary.LittleEndian.Uint32(src[4:])
			for j := 0; j < 16; j++ {
				q := src[8+j]
				x0 := int(q&0xf) | int((qh>>uint(j))&1)<<4
				x1 := int(q>>4) | int((qh>>uint(j+16))&1)<<4
				dst[j] = float32(x0)*d + m
				dst[j+16] = float32(x1)*d + m
			}
		case Q8_0:
			d := f16(src)
			for j := 0; j < 32; j++ {
				dst[j] = float32(int8(src[2+j])) * d
			}
		}
	}
	return out, nil
}
