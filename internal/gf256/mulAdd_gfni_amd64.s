#include "textflag.h"

// func mulAddGFNI(dst, src []byte, matrix uint64)
// AVX512 GFNI: VGF2P8AFFINEQB 单指令GF(2^8)乘法
TEXT ·mulAddGFNI(SB), NOSPLIT, $0-56
	MOVQ	dst_base+0(FP), DI
	MOVQ	dst_len+8(FP), CX
	MOVQ	src_base+24(FP), SI
	MOVQ	matrix+48(FP), AX

	// broadcast affine matrix到ZMM15
	VPBROADCASTQ AX, Z15

loop64:
	CMPQ	CX, $64
	JL	done

	VMOVDQU64	(SI), Z0
	// VGF2P8AFFINEQB $0, Z15, Z0, Z1
	BYTE $0x62; BYTE $0xF3; BYTE $0x85; BYTE $0x48
	BYTE $0xCE; BYTE $0xC8; BYTE $0x00

	VMOVDQU64	(DI), Z2
	VPXORQ	Z1, Z2, Z2
	VMOVDQU64	Z2, (DI)

	ADDQ	$64, SI
	ADDQ	$64, DI
	SUBQ	$64, CX
	JMP	loop64

done:
	VZEROUPPER
	RET

// func mulGFNINoXor(dst, src []byte, matrix uint64)
// GFNI multiply without XOR: dst = src * matrix
TEXT ·mulGFNINoXor(SB), NOSPLIT, $0-56
	MOVQ	dst_base+0(FP), DI
	MOVQ	dst_len+8(FP), CX
	MOVQ	src_base+24(FP), SI
	MOVQ	matrix+48(FP), AX

	VPBROADCASTQ AX, Z15

loop64_noxor:
	CMPQ	CX, $64
	JL	done_noxor

	VMOVDQU64	(SI), Z0
	// VGF2P8AFFINEQB $0, Z15, Z0, Z1
	BYTE $0x62; BYTE $0xF3; BYTE $0x85; BYTE $0x48
	BYTE $0xCE; BYTE $0xC8; BYTE $0x00

	VMOVDQU64	Z1, (DI)

	ADDQ	$64, SI
	ADDQ	$64, DI
	SUBQ	$64, CX
	JMP	loop64_noxor

done_noxor:
	VZEROUPPER
	RET
