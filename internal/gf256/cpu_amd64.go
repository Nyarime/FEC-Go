//go:build amd64

package gf256

import "golang.org/x/sys/cpu"

// CPU feature flags for dispatch
var (
	hasGFNI    = cpu.X86.HasAVX512 && cpu.X86.HasAVX512VL
	hasAVX2    = cpu.X86.HasAVX2
	hasSSE41   = cpu.X86.HasSSE41
	hasBMI2    = cpu.X86.HasBMI2

	// AMD Zen1/Zen2 has BMI2 but PEXT/PDEP are microcoded (slow)
	// Zen3+ fixed this. We detect via AVX512 support (Zen4+) or just avoid BMI2 path.
)

// GFNI affine变换矩阵: 每个GF(2^8)系数对应一个8x8位矩阵
var gfniMatrix [256]uint64

func init() {
	for c := 0; c < 256; c++ {
		var m uint64
		for bit := 0; bit < 8; bit++ {
			row := Mul(byte(c), 1<<bit)
			m |= uint64(row) << (bit * 8)
		}
		gfniMatrix[c] = m
	}
}

// CPUInfo returns a string describing the detected SIMD capabilities
func CPUInfo() string {
	switch {
	case hasGFNI:
		return "AVX-512 GFNI (64B/op, single-instruction GF multiply)"
	case hasAVX2:
		return "AVX2 VPSHUFB (32B/op, split table lookup)"
	case hasSSE41:
		return "SSE4.1 PSHUFB (16B/op, split table lookup)"
	default:
		return "Scalar (1B/op)"
	}
}
