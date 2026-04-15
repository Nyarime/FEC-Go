//go:build arm64

package gf256

//go:noescape
func mulAddNEON(dst, src []byte, loTable, hiTable *[16]byte)

// MulAddRegion dst ^= src * coeff (ARM64 NEON加速)
func MulAddRegion(dst, src []byte, coeff byte) {
	if coeff == 0 { return }
	if coeff == 1 {
		for i := range dst {
			if i < len(src) { dst[i] ^= src[i] }
		}
		return
	}

	var loTable, hiTable [16]byte
	for i := byte(0); i < 16; i++ {
		loTable[i] = Mul(i, coeff)
		hiTable[i] = Mul(i<<4, coeff)
	}

	n := len(dst)
	if len(src) < n { n = len(src) }

	aligned := n &^ 15 // 16字节对齐
	if aligned > 0 {
		mulAddNEON(dst[:aligned], src[:aligned], &loTable, &hiTable)
	}
	for i := aligned; i < n; i++ {
		dst[i] ^= Mul(src[i], coeff)
	}
}
