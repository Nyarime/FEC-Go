//go:build amd64

package gf256

//go:noescape
func mulAddAVX2(dst, src []byte, loTable, hiTable *[16]byte)

// MulAddRegion dst ^= src * coeff (AVX2 VPSHUFB加速)
func MulAddRegion(dst, src []byte, coeff byte) {
	if coeff == 0 { return }
	if coeff == 1 {
		for i := range dst {
			if i < len(src) { dst[i] ^= src[i] }
		}
		return
	}

	n := len(dst)
	if len(src) < n { n = len(src) }

	aligned := n &^ 31
	if aligned > 0 {
		mulAddAVX2(dst[:aligned], src[:aligned], &mulLo[coeff], &mulHi[coeff])
	}
	for i := aligned; i < n; i++ {
		dst[i] ^= Mul(src[i], coeff)
	}
}
