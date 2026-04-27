//go:build !amd64

package gf65536

import "unsafe"

// MulAddRegion computes dst[i:i+2] ^= src[i:i+2] * coeff for each 16-bit element.
// dst and src must have equal length (multiple of 2).
// This is the hot path for Reed-Solomon encoding/decoding.
func MulAddRegion(dst, src []byte, coeff Element) {
	if len(dst) != len(src) {
		panic("gf65536: MulAddRegion dst/src length mismatch")
	}
	if coeff == 0 {
		return
	}
	if coeff == 1 {
		for i := range dst {
			dst[i] ^= src[i]
		}
		return
	}

	n := len(dst) / 2
	if n == 0 {
		return
	}
	dstW := unsafe.Slice((*uint16)(unsafe.Pointer(&dst[0])), n)
	srcW := unsafe.Slice((*uint16)(unsafe.Pointer(&src[0])), n)

	for i := 0; i < n; i++ {
		dstW[i] ^= uint16(Mul(Element(srcW[i]), coeff))
	}
}

// MulRegion computes dst[i:i+2] = src[i:i+2] * coeff for each 16-bit element.
func MulRegion(dst, src []byte, coeff Element) {
	if len(dst) != len(src) {
		panic("gf65536: MulRegion dst/src length mismatch")
	}
	if coeff == 0 {
		for i := range dst {
			dst[i] = 0
		}
		return
	}
	if coeff == 1 {
		copy(dst, src)
		return
	}

	n := len(dst) / 2
	if n == 0 {
		return
	}
	dstW := unsafe.Slice((*uint16)(unsafe.Pointer(&dst[0])), n)
	srcW := unsafe.Slice((*uint16)(unsafe.Pointer(&src[0])), n)

	for i := 0; i < n; i++ {
		dstW[i] = uint16(Mul(Element(srcW[i]), coeff))
	}
}
