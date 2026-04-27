package gf65536

import (
	"encoding/binary"
	"math/rand/v2"
	"testing"
)

func TestTableInit(t *testing.T) {
	// generator^0 = 1
	if expTable[0] != 1 {
		t.Fatalf("exp[0] = %d, want 1", expTable[0])
	}
	// generator^1 = 2
	if expTable[1] != Element(Generator) {
		t.Fatalf("exp[1] = %d, want %d", expTable[1], Generator)
	}
	// log(1) = 0
	if logTable[1] != 0 {
		t.Fatalf("log[1] = %d, want 0", logTable[1])
	}
	// All nonzero elements appear exactly once in expTable[0..Order-2]
	seen := make(map[Element]bool)
	for i := 0; i < Order-1; i++ {
		v := expTable[i]
		if v == 0 {
			t.Fatalf("exp[%d] = 0", i)
		}
		if seen[v] {
			t.Fatalf("exp[%d] = %d duplicate", i, v)
		}
		seen[v] = true
	}
	if len(seen) != Order-1 {
		t.Fatalf("only %d unique elements, want %d", len(seen), Order-1)
	}
}

func TestMulIdentity(t *testing.T) {
	for a := Element(0); a < 256; a++ {
		if Mul(a, 1) != a {
			t.Fatalf("Mul(%d, 1) = %d", a, Mul(a, 1))
		}
		if Mul(a, 0) != 0 {
			t.Fatalf("Mul(%d, 0) = %d", a, Mul(a, 0))
		}
	}
}

func TestMulInv(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	for i := 0; i < 10000; i++ {
		a := Element(rng.UintN(65535)) + 1 // nonzero
		inv := Inv(a)
		if Mul(a, inv) != 1 {
			t.Fatalf("Mul(%d, Inv(%d)) = %d, want 1", a, a, Mul(a, inv))
		}
	}
}

func TestDivMul(t *testing.T) {
	rng := rand.New(rand.NewPCG(99, 0))
	for i := 0; i < 10000; i++ {
		a := Element(rng.UintN(65536))
		b := Element(rng.UintN(65535)) + 1
		c := Mul(a, b)
		if Div(c, b) != a {
			t.Fatalf("Div(Mul(%d,%d), %d) = %d, want %d", a, b, b, Div(c, b), a)
		}
	}
}

func TestAssociativity(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 0))
	for i := 0; i < 10000; i++ {
		a := Element(rng.UintN(65536))
		b := Element(rng.UintN(65536))
		c := Element(rng.UintN(65536))
		if Mul(Mul(a, b), c) != Mul(a, Mul(b, c)) {
			t.Fatalf("associativity failed for %d,%d,%d", a, b, c)
		}
	}
}

func TestDistributivity(t *testing.T) {
	rng := rand.New(rand.NewPCG(13, 0))
	for i := 0; i < 10000; i++ {
		a := Element(rng.UintN(65536))
		b := Element(rng.UintN(65536))
		c := Element(rng.UintN(65536))
		lhs := Mul(a, Add(b, c))
		rhs := Add(Mul(a, b), Mul(a, c))
		if lhs != rhs {
			t.Fatalf("distributivity failed for %d,%d,%d: %d != %d", a, b, c, lhs, rhs)
		}
	}
}

func TestExpLog(t *testing.T) {
	for i := 0; i < Order-1; i++ {
		e := Exp(i)
		if Log(e) != i {
			t.Fatalf("Log(Exp(%d)) = %d", i, Log(e))
		}
	}
}

func TestMulAddRegion(t *testing.T) {
	rng := rand.New(rand.NewPCG(55, 0))
	n := 128
	src := make([]byte, n*2)
	dst := make([]byte, n*2)
	expect := make([]byte, n*2)

	for i := range src {
		src[i] = byte(rng.UintN(256))
	}
	for i := range dst {
		dst[i] = byte(rng.UintN(256))
		expect[i] = dst[i]
	}

	coeff := Element(rng.UintN(65535)) + 1
	// compute expected
	for i := 0; i < n; i++ {
		s := Element(binary.LittleEndian.Uint16(src[i*2:]))
		d := Element(binary.LittleEndian.Uint16(expect[i*2:]))
		binary.LittleEndian.PutUint16(expect[i*2:], uint16(Add(d, Mul(s, coeff))))
	}

	MulAddRegion(dst, src, coeff)

	for i := range dst {
		if dst[i] != expect[i] {
			t.Fatalf("MulAddRegion mismatch at byte %d: got %d, want %d", i, dst[i], expect[i])
		}
	}
}

func BenchmarkMul(b *testing.B) {
	a, c := Element(12345), Element(54321)
	for i := 0; i < b.N; i++ {
		a = Mul(a, c)
	}
	_ = a
}

func BenchmarkMulAddRegion(b *testing.B) {
	n := 4096
	src := make([]byte, n)
	dst := make([]byte, n)
	coeff := Element(42)
	b.SetBytes(int64(n))
	for i := 0; i < b.N; i++ {
		MulAddRegion(dst, src, coeff)
	}
}
