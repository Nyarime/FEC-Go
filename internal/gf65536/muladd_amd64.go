//go:build amd64

package gf65536

import "unsafe"

// mulAddRegionTable threshold: only use 64KB precomputed table for buffers >= this size.
// Below this, per-element log/exp is faster due to table init cost.
const tableThreshold = 8192 // bytes (4096 uint16 elements)

// MulAddRegion computes dst[i] ^= src[i] * coeff for each 16-bit GF(2^16) element.
func MulAddRegion(dst, src []byte, coeff Element) {
	if len(dst) != len(src) {
		panic("gf65536: MulAddRegion dst/src length mismatch")
	}
	if coeff == 0 {
		return
	}
	if coeff == 1 {
		n8 := len(dst) / 8
		if n8 > 0 {
			dstW := unsafe.Slice((*uint64)(unsafe.Pointer(&dst[0])), n8)
			srcW := unsafe.Slice((*uint64)(unsafe.Pointer(&src[0])), n8)
			for i := range dstW {
				dstW[i] ^= srcW[i]
			}
		}
		for i := n8 * 8; i < len(dst); i++ {
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

	if len(dst) >= tableThreshold {
		mulAddRegionTable(dstW, srcW, coeff)
	} else {
		mulAddRegionScalar(dstW, srcW, coeff)
	}
}

func mulAddRegionScalar(dstW, srcW []uint16, coeff Element) {
	for i := range dstW {
		dstW[i] ^= uint16(Mul(Element(srcW[i]), coeff))
	}
}

func mulAddRegionTable(dstW, srcW []uint16, coeff Element) {
	var mulTab [65536]uint16
	logC := logTable[coeff]
	for x := 1; x < 65536; x++ {
		logSum := int(logTable[x]) + int(logC)
		if logSum >= Order-1 {
			logSum -= Order - 1
		}
		mulTab[x] = uint16(expTable[logSum])
	}

	n := len(dstW)
	n4 := n &^ 3
	for i := 0; i < n4; i += 4 {
		dstW[i] ^= mulTab[srcW[i]]
		dstW[i+1] ^= mulTab[srcW[i+1]]
		dstW[i+2] ^= mulTab[srcW[i+2]]
		dstW[i+3] ^= mulTab[srcW[i+3]]
	}
	for i := n4; i < n; i++ {
		dstW[i] ^= mulTab[srcW[i]]
	}
}

// MulRegion computes dst[i] = src[i] * coeff for each 16-bit element.
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

	if len(dst) >= tableThreshold {
		var mulTab [65536]uint16
		logC := logTable[coeff]
		for x := 1; x < 65536; x++ {
			logSum := int(logTable[x]) + int(logC)
			if logSum >= Order-1 {
				logSum -= Order - 1
			}
			mulTab[x] = uint16(expTable[logSum])
		}
		for i := 0; i < n; i++ {
			dstW[i] = mulTab[srcW[i]]
		}
	} else {
		for i := 0; i < n; i++ {
			dstW[i] = uint16(Mul(Element(srcW[i]), coeff))
		}
	}
}
