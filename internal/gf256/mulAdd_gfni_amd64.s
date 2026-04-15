#include "textflag.h"

// func mulAddGFNI(dst, src []byte, matrix uint64)
// AVX512 GFNI: VGF2P8AFFINEQB 单指令GF(2^8)乘法
// 64字节/次 (ZMM寄存器)
TEXT ·mulAddGFNI(SB), NOSPLIT, $0-40
	MOVQ	dst_base+0(FP), DI
	MOVQ	dst_len+8(FP), CX
	MOVQ	src_base+24(FP), SI
	// matrix是GF(2^8)乘法的affine变换矩阵
	MOVQ	matrix+48(FP), AX

	// broadcast matrix到ZMM15
	VPBROADCASTQ AX, Z15

loop64:
	CMPQ	CX, $64
	JL	loop32

	// 加载64字节src
	VMOVDQU64	(SI), Z0
	// GF(2^8) affine变换: dst = matrix * src (mod poly)
	// VGF2P8AFFINEQB imm8=0, Z15, Z0, Z1
	BYTE $0x62; BYTE $0xF3; BYTE $0x85; BYTE $0x48
	BYTE $0xCE; BYTE $0xC8; BYTE $0x00
	// Z1 ^= dst
	VMOVDQU64	(DI), Z2
	VPXORQ	Z1, Z2, Z2
	VMOVDQU64	Z2, (DI)

	ADDQ	$64, SI
	ADDQ	$64, DI
	SUBQ	$64, CX
	JMP	loop64

loop32:
	// fallback到AVX2路径(由Go层处理)
	VZEROUPPER
	RET
