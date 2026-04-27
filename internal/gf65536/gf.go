// Package gf65536 implements GF(2^16) finite field arithmetic.
// Irreducible polynomial: x^16 + x^12 + x^3 + x + 1 (0x1002B).
package gf65536

// Element is a GF(2^16) field element (0..65535).
type Element uint16

const (
	Order     = 65536
	Modulus   = 0x1002B // x^16 + x^12 + x^3 + x + 1
	Generator = 3       // primitive element
)

// Add returns a + b in GF(2^16) (XOR).
func Add(a, b Element) Element { return a ^ b }

// Sub returns a - b in GF(2^16) (same as Add in characteristic 2).
func Sub(a, b Element) Element { return a ^ b }

// Mul returns a * b in GF(2^16) using log/exp tables.
func Mul(a, b Element) Element {
	if a == 0 || b == 0 {
		return 0
	}
	logSum := int(logTable[a]) + int(logTable[b])
	if logSum >= Order-1 {
		logSum -= Order - 1
	}
	return expTable[logSum]
}

// Div returns a / b in GF(2^16).
func Div(a, b Element) Element {
	if b == 0 {
		panic("gf65536: division by zero")
	}
	if a == 0 {
		return 0
	}
	logDiff := int(logTable[a]) - int(logTable[b])
	if logDiff < 0 {
		logDiff += Order - 1
	}
	return expTable[logDiff]
}

// Inv returns 1/a in GF(2^16).
func Inv(a Element) Element {
	if a == 0 {
		panic("gf65536: inverse of zero")
	}
	return expTable[Order-1-int(logTable[a])]
}

// Exp returns generator^n in GF(2^16).
func Exp(n int) Element {
	n %= (Order - 1)
	if n < 0 {
		n += Order - 1
	}
	return expTable[n]
}

// Log returns log_generator(a).
func Log(a Element) int {
	if a == 0 {
		panic("gf65536: log of zero")
	}
	return int(logTable[a])
}
