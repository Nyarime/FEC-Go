//go:build amd64

package gf256

//go:noescape
func mulAddAVX2(dst, src []byte, loTable, hiTable *[16]byte)

//go:noescape
func mulAddGFNI(dst, src []byte, matrix uint64)

// MulAddRegion dst ^= src * coeff
// 自动选择: GFNI(64B/op) > AVX2(32B/op) > scalar
func MulAddRegion(dst, src []byte, coeff byte) {
	if coeff == 0 { return }

	n := len(dst)
	if len(src) < n { n = len(src) }

	if coeff == 1 {
		for i := 0; i < n; i++ { dst[i] ^= src[i] }
		return
	}

	if hasGFNI {
		aligned := n &^ 63
		if aligned > 0 {
			mulAddGFNI(dst[:aligned], src[:aligned], gfniMatrix[coeff])
		}
		for i := aligned; i < n; i++ {
			dst[i] ^= Mul(src[i], coeff)
		}
		return
	}

	if hasAVX2 {
		aligned := n &^ 31
		if aligned > 0 {
			mulAddAVX2(dst[:aligned], src[:aligned], &mulLo[coeff], &mulHi[coeff])
		}
		for i := aligned; i < n; i++ {
			dst[i] ^= Mul(src[i], coeff)
		}
		return
	}

	// SSE4.1 / scalar fallback
	for i := 0; i < n; i++ {
		dst[i] ^= Mul(src[i], coeff)
	}
}

// MulRegion dst = src * coeff (overwrite, not XOR)
func MulRegion(dst, src []byte, coeff byte) {
	n := len(dst)
	if len(src) < n { n = len(src) }

	if coeff == 0 {
		for i := 0; i < n; i++ { dst[i] = 0 }
		return
	}
	if coeff == 1 {
		copy(dst[:n], src[:n])
		return
	}

	// GFNI: copy src*coeff to dst (no XOR)
	if hasGFNI {
		aligned := n &^ 63
		if aligned > 0 {
			mulGFNINoXor(dst[:aligned], src[:aligned], gfniMatrix[coeff])
		}
		for i := aligned; i < n; i++ {
			dst[i] = Mul(src[i], coeff)
		}
		return
	}

	// Scalar
	for i := 0; i < n; i++ {
		dst[i] = Mul(src[i], coeff)
	}
}

//go:noescape
func mulGFNINoXor(dst, src []byte, matrix uint64)
