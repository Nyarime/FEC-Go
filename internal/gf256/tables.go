package gf256

// 预计算split tables: mulLo[coeff][nibble], mulHi[coeff][nibble]
// 8KB总量, 启动时一次性生成, L1 cache友好
var (
	mulLo [256][16]byte
	mulHi [256][16]byte
)

func init() {
	for c := 0; c < 256; c++ {
		for i := byte(0); i < 16; i++ {
			mulLo[c][i] = Mul(byte(c), i)
			mulHi[c][i] = Mul(byte(c), i<<4)
		}
	}
}
