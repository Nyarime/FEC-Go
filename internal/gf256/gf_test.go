package gf256

import (
	"runtime"
	"testing"
)

func TestMul(t *testing.T) {
	// 基本性质
	if Mul(0, 5) != 0 { t.Fatal("0*x != 0") }
	if Mul(1, 5) != 5 { t.Fatal("1*x != x") }
	if Mul(2, 3) != Mul(3, 2) { t.Fatal("交换律失败") }
	t.Log("✅ GF(256)乘法基本性质通过")

	// 逆元
	for a := byte(1); a != 0; a++ {
		found := false
		for b := byte(1); b != 0; b++ {
			if Mul(a, b) == 1 {
				found = true
				break
			}
		}
		if !found { t.Fatalf("❌ %d没有逆元", a) }
	}
	t.Log("✅ 所有非零元素都有逆元")
}

func TestMulAddSplit(t *testing.T) {
	dst1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	dst2 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	src := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	MulAdd(dst1, src, 42)
	MulAddSplit(dst2, src, 42)

	for i := range dst1 {
		if dst1[i] != dst2[i] {
			t.Fatalf("❌ 位置%d不匹配: MulAdd=%d MulAddSplit=%d", i, dst1[i], dst2[i])
		}
	}
	t.Log("✅ MulAdd和MulAddSplit结果一致")
}

func BenchmarkMul(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Mul(byte(i), byte(i+1))
	}
}

func BenchmarkMulAdd(b *testing.B) {
	dst := make([]byte, 4096)
	src := make([]byte, 4096)
	b.SetBytes(4096)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MulAdd(dst, src, 42)
	}
}

func BenchmarkMulAddSplit(b *testing.B) {
	dst := make([]byte, 4096)
	src := make([]byte, 4096)
	b.SetBytes(4096)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MulAddSplit(dst, src, 42)
	}
}

func BenchmarkMulAddRegion(b *testing.B) {
	sizes := []struct{ name string; n int }{
		{"256B", 256},
		{"1KB", 1024},
		{"4KB", 4096},
		{"32KB", 32768},
	}
	for _, s := range sizes {
		dst := make([]byte, s.n)
		src := make([]byte, s.n)
		for i := range src { src[i] = byte(i) }
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.n))
			for i := 0; i < b.N; i++ {
				MulAddRegion(dst, src, 42)
			}
		})
	}
}



func TestCPUFeatures(t *testing.T) {
	t.Logf("GOARCH: %s", runtime.GOARCH)
	if runtime.GOARCH == "amd64" {
		t.Logf("GFNI available: %v", hasGFNI)
	}
	t.Log("✅ CPU特性检测正常")
}
