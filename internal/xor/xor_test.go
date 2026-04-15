package xor

import "testing"

func BenchmarkXOR(b *testing.B) {
	sizes := []struct{ name string; n int }{
		{"64B", 64},
		{"256B", 256},
		{"1KB", 1024},
		{"4KB", 4096},
		{"32KB", 32768},
		{"1MB", 1048576},
	}

	for _, s := range sizes {
		dst := make([]byte, s.n)
		src := make([]byte, s.n)
		for i := range src { src[i] = byte(i) }

		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.n))
			for i := 0; i < b.N; i++ {
				Bytes(dst, src)
			}
		})
	}
}
