package ldpc

import (
	"testing"
	"bytes"
)

func TestLDPCEncodeDecode(t *testing.T) {
	codec := New(8, 4, 0.3) // 8数据+4校验
	
	// 原始数据
	data := make([][]byte, 8)
	for i := range data {
		data[i] = []byte{byte(i*10+1), byte(i*10+2), byte(i*10+3), byte(i*10+4)}
	}
	
	// 编码
	encoded := codec.Encode(data)
	if len(encoded) != 12 {
		t.Fatalf("expected 12 shards, got %d", len(encoded))
	}
	t.Logf("✅ 编码: %d数据 + %d校验 = %d总", 8, 4, len(encoded))
	
	// 模拟丢失2个数据块
	encoded[2] = nil
	encoded[5] = nil
	t.Log("模拟丢失: shard[2], shard[5]")
	
	// 解码
	err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("❌ 解码失败: %v", err)
	}
	
	// 验证恢复
	if !bytes.Equal(encoded[2], data[2]) {
		t.Errorf("❌ shard[2] 恢复错误: got %v want %v", encoded[2], data[2])
	} else {
		t.Log("✅ shard[2] 恢复成功")
	}
	if !bytes.Equal(encoded[5], data[5]) {
		t.Errorf("❌ shard[5] 恢复错误: got %v want %v", encoded[5], data[5])
	} else {
		t.Log("✅ shard[5] 恢复成功")
	}
}

func TestLDPCSingleLoss(t *testing.T) {
	codec := New(4, 2, 0.5) // 高密度
	
	data := [][]byte{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
		{13, 14, 15, 16},
	}
	
	encoded := codec.Encode(data)
	original := make([]byte, 4)
	copy(original, encoded[1])
	
	// 丢1个
	encoded[1] = nil
	
	err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("❌ 单块丢失解码失败: %v", err)
	}
	
	if bytes.Equal(encoded[1], original) {
		t.Log("✅ 单块丢失恢复成功")
	} else {
		t.Errorf("❌ 恢复不匹配: got %v want %v", encoded[1], original)
	}
}

func BenchmarkLDPCEncode(b *testing.B) {
	codec := New(16, 8, 0.3)
	data := make([][]byte, 16)
	for i := range data {
		data[i] = make([]byte, 1024) // 1KB块
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Encode(data)
	}
}

func BenchmarkLDPCDecode(b *testing.B) {
	codec := New(16, 8, 0.3)
	data := make([][]byte, 16)
	for i := range data {
		data[i] = make([]byte, 1024)
		for j := range data[i] { data[i][j] = byte(i*j) }
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded := codec.Encode(data)
		encoded[3] = nil
		encoded[7] = nil
		codec.Decode(encoded)
	}
}
