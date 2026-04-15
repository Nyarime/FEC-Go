//go:build amd64

package gf256

//go:noescape
func mulAddAVX2(dst, src []byte, loTable, hiTable *[16]byte)

// GF(2^8)乘法的affine矩阵(用于GFNI VGF2P8AFFINEQB)
var gfniMatrix [256]uint64

func init() {
	// 预计算每个系数的affine变换矩阵
	for c := 0; c < 256; c++ {
		var m uint64
		for bit := 0; bit < 8; bit++ {
			// 每一行是c乘以basis向量的结果
			row := Mul(byte(c), 1<<bit)
			m |= uint64(row) << (bit * 8)
		}
		gfniMatrix[c] = m
	}
}

// MulAddRegion dst ^= src * coeff
// 自动选择: GFNI(64B/op) > AVX2 VPSHUFB(32B/op) > scalar
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

	// 暂时统一走AVX2(GFNI需要AVX512支持的CPU)
	// TODO: hasGFNI时走mulAddGFNI
	aligned := n &^ 31
	if aligned > 0 {
		mulAddAVX2(dst[:aligned], src[:aligned], &mulLo[coeff], &mulHi[coeff])
	}
	for i := aligned; i < n; i++ {
		dst[i] ^= Mul(src[i], coeff)
	}
}
