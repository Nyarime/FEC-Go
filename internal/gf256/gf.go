// Package gf256 provides GF(2^8) arithmetic with lookup table acceleration.
// Primitive polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x11D)
package gf256

// 预计算log/exp表(初始化时一次性生成)
var logTable [256]byte
var expTable [512]byte // 512 for wraparound

func init() {
	// 生成log/exp表
	x := byte(1)
	for i := 0; i < 255; i++ {
		expTable[i] = x
		expTable[i+255] = x
		logTable[x] = byte(i)
		// x *= 2 in GF(2^8) with primitive poly 0x1D
		if x&0x80 != 0 {
			x = (x << 1) ^ 0x1D
		} else {
			x <<= 1
		}
	}
	logTable[0] = 0
}

// Mul GF(256)乘法(查表, O(1))
func Mul(a, b byte) byte {
	if a == 0 || b == 0 { return 0 }
	return expTable[int(logTable[a])+int(logTable[b])]
}

// MulAdd dst[i] ^= src[i] * coeff (标量版)
func MulAdd(dst, src []byte, coeff byte) {
	if coeff == 0 { return }
	if coeff == 1 {
		for i := range dst {
			if i < len(src) { dst[i] ^= src[i] }
		}
		return
	}
	for i := range dst {
		if i < len(src) {
			dst[i] ^= Mul(src[i], coeff)
		}
	}
}

// MulAddSplit 用split-table加速MulAdd(16字节批量)
// lo/hi nibble分别查表+XOR
func MulAddSplit(dst, src []byte, coeff byte) {
	if coeff == 0 { return }
	if coeff == 1 {
		for i := range dst {
			if i < len(src) { dst[i] ^= src[i] }
		}
		return
	}

	// 预计算split tables
	var loTable, hiTable [16]byte
	for i := byte(0); i < 16; i++ {
		loTable[i] = Mul(i, coeff)
		hiTable[i] = Mul(i<<4, coeff)
	}

	// 查表XOR
	for i := range dst {
		if i < len(src) {
			lo := src[i] & 0x0F
			hi := src[i] >> 4
			dst[i] ^= loTable[lo] ^ hiTable[hi]
		}
	}
}
