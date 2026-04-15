#include "textflag.h"

// func mulAddNEON(dst, src []byte, loTable, hiTable *[16]byte)
// ARM64 NEON: VTBL nibble lookup + VEOR
TEXT ·mulAddNEON(SB), NOSPLIT, $0-56
	MOVD	dst_base+0(FP), R0     // dst
	MOVD	dst_len+8(FP), R2      // dst len
	MOVD	src_base+24(FP), R1    // src
	MOVD	src_len+32(FP), R3     // src len
	MOVD	loTable+48(FP), R4     // lo table
	MOVD	hiTable+56(FP), R5     // hi table

	// min(len)
	CMP	R3, R2
	CSEL	LT, R2, R3, R2

	// 加载lo/hi lookup table到NEON寄存器
	VLD1	(R4), [V30.B16]        // lo table
	VLD1	(R5), [V31.B16]        // hi table

	// nibble mask = 0x0F
	VMOVI	$0x0F, V29.B16

loop16:
	CMP	$16, R2
	BLT	tail

	VLD1	(R1), [V0.B16]         // src 16B
	VAND	V0.B16, V29.B16, V1.B16  // lo nibble
	VUSHR	$4, V0.B16, V2.B16    // hi nibble

	VTBL	V1.B16, [V30.B16], V1.B16  // lo lookup
	VTBL	V2.B16, [V31.B16], V2.B16  // hi lookup
	VEOR	V1.B16, V2.B16, V1.B16    // gf_mul result

	VLD1	(R0), [V3.B16]         // dst
	VEOR	V1.B16, V3.B16, V3.B16 // dst ^= result
	VST1	[V3.B16], (R0)

	ADD	$16, R0
	ADD	$16, R1
	SUB	$16, R2
	B	loop16

tail:
	// Go层处理剩余
	RET
