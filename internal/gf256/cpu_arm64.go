//go:build arm64

package gf256

import "golang.org/x/sys/cpu"

var (
	hasNEON    = true // AArch64 always has NEON
	hasAESArm  = cpu.ARM64.HasAES
	hasSHA2Arm = cpu.ARM64.HasSHA2
	hasSVE     = cpu.ARM64.HasSVE
)

// CPUInfo returns a string describing the detected SIMD capabilities
func CPUInfo() string {
	extra := ""
	if hasAESArm { extra += " +AES" }
	if hasSHA2Arm { extra += " +SHA2" }
	if hasSVE { return "SVE (variable-width SIMD)" + extra }
	return "NEON (128-bit SIMD)" + extra
}
