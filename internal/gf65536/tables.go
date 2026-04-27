package gf65536

var (
	expTable [Order]Element // expTable[i] = generator^i
	logTable [Order]Element // logTable[a] = i where generator^i = a
)

// mulGF multiplies two elements in GF(2^16) using polynomial arithmetic.
func mulGF(a, b uint32) uint32 {
	var r uint32
	for b > 0 {
		if b&1 != 0 {
			r ^= a
		}
		a <<= 1
		if a >= uint32(Order) {
			a ^= Modulus
		}
		b >>= 1
	}
	return r
}

func init() {
	var val uint32 = 1
	for i := 0; i < Order-1; i++ {
		expTable[i] = Element(val)
		logTable[val] = Element(i)
		val = mulGF(val, Generator)
	}
	expTable[Order-1] = expTable[0] // wrap around
}
