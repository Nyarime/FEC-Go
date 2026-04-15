//go:build amd64

package gf256

import "golang.org/x/sys/cpu"

var hasGFNI = cpu.X86.HasAVX512 && cpu.X86.HasAVX512VL

func init() {
	// Broadwell等老CPU: hasGFNI=false, 走AVX2 VPSHUFB
	// Ice Lake/Zen4+: hasGFNI=true, 走GFNI单指令乘法
}
