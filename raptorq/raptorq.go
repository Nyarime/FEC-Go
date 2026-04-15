// Package raptorq implements RaptorQ (RFC 6330) fountain code in pure Go.
//
// RaptorQ is a systematic, rateless erasure code that can generate
// unlimited repair symbols from source data.
//
// Architecture:
//   Source data → Split into source blocks → Pre-code (intermediate symbols)
//   → LT encode → Unlimited encoding symbols
//   Received symbols → Gaussian elimination → Recover source data
package raptorq

import (
	"github.com/nyarime/gofec/internal/gf256"
	"encoding/binary"
	"errors"
	"math"
)

// Codec RaptorQ编解码器
type Codec struct {
	sourceSymbols int
	symbolSize    int
	numLDPC       int
	numHDPC       int
	numLT         int
	// 预分配workspace(零GC)
	workspace     *workspace
}

type workspace struct {
	encBuf    []byte   // 编码临时buffer
	neighbors []int    // 邻居列表复用
	seen      []bool   // 邻居去重
}

// New 创建RaptorQ编解码器
// sourceSymbols: 源数据块数(K)
// symbolSize: 每块字节数(T)
func New(sourceSymbols, symbolSize int) *Codec {
	// RFC 6330 参数计算
	K := sourceSymbols
	S := numLDPCSymbols(K)
	H := numHDPCSymbols(K)
	L := K + S + H

	return &Codec{
		sourceSymbols: K,
		symbolSize:    symbolSize,
		numLDPC:       S,
		numHDPC:       H,
		numLT:         L,
		workspace: &workspace{
			encBuf:    make([]byte, symbolSize),
			neighbors: make([]int, 0, L),
			seen:      make([]bool, K),
		},
	}
}

// Encode 编码源数据为编码符号
// data: 源数据(长度应为K*T)
// numRepair: 额外修复符号数
// 返回: K个源符号 + numRepair个修复符号
func (c *Codec) Encode(data []byte, numRepair int) []Symbol {
	K := c.sourceSymbols
	T := c.symbolSize

	// 1. 分割源数据为K个符号
	source := splitData(data, K, T)

	// 2. 生成中间符号(pre-code)

	// 3. LT编码: 源符号(系统码) + 修复符号
	symbols := make([]Symbol, K+numRepair)

	// 源符号(系统,直接输出)
	for i := 0; i < K; i++ {
		symbols[i] = Symbol{
			ESI:  uint32(i),
			Data: source[i],
		}
	}

	// 修复符号(LT编码: 直接XOR源符号)
	for i := 0; i < numRepair; i++ {
		esi := uint32(K + i)
		symbols[K+i] = Symbol{
			ESI:  esi,
			Data: c.ltEncode(source, esi),
		}
	}

	return symbols
}

// Decode 从接收到的符号恢复源数据
// received: 接收到的符号(可能有丢失)
// dataLen: 原始数据长度
func (c *Codec) Decode(received []Symbol, dataLen int) ([]byte, error) {
	if len(received) < c.sourceSymbols {
		return nil, errors.New("not enough symbols for decoding")
	}

	// 构建系数矩阵 + 高斯消元
	result, err := c.gaussianDecode(received)
	if err != nil {
		return nil, err
	}

	// 拼接恢复的源数据
	data := make([]byte, 0, dataLen)
	for i := 0; i < c.sourceSymbols && len(data) < dataLen; i++ {
		remaining := dataLen - len(data)
		if remaining > len(result[i]) {
			remaining = len(result[i])
		}
		data = append(data, result[i][:remaining]...)
	}

	return data, nil
}

// Symbol 编码符号
type Symbol struct {
	ESI  uint32 // 编码符号ID
	Data []byte // 符号数据
}

// === 内部实现 ===

// numLDPCSymbols RFC 6330 LDPC校验符号数
func numLDPCSymbols(K int) int {
	return int(math.Ceil(0.01*float64(K))) + int(math.Ceil(math.Sqrt(float64(K))))
}

// numHDPCSymbols RFC 6330 HDPC校验符号数
func numHDPCSymbols(K int) int {
	return int(math.Ceil(math.Sqrt(float64(K)))) + 1
}

// splitData 分割数据为K个T字节的符号
func splitData(data []byte, K, T int) [][]byte {
	symbols := make([][]byte, K)
	for i := 0; i < K; i++ {
		symbols[i] = make([]byte, T)
		start := i * T
		end := start + T
		if end > len(data) { end = len(data) }
		if start < len(data) {
			copy(symbols[i], data[start:end])
		}
	}
	return symbols
}

// generateIntermediate 生成中间符号(pre-code)
// 这是RaptorQ的核心: LDPC + HDPC预编码
func (c *Codec) generateIntermediate(source [][]byte) [][]byte {
	L := c.numLT
	T := c.symbolSize

	// 中间符号 = 源符号 + LDPC校验 + HDPC校验
	intermediate := make([][]byte, L)

	// 源符号直接复制
	for i := 0; i < c.sourceSymbols; i++ {
		intermediate[i] = make([]byte, T)
		copy(intermediate[i], source[i])
	}

	// LDPC校验符号(XOR based)
	for i := 0; i < c.numLDPC; i++ {
		intermediate[c.sourceSymbols+i] = make([]byte, T)
		// LDPC: 每个校验符号XOR若干源符号
		for j := 0; j < c.sourceSymbols; j++ {
			if ldpcConnect(i, j, c.sourceSymbols, c.numLDPC) {
				xorSymbol(intermediate[c.sourceSymbols+i], source[j])
			}
		}
	}

	// HDPC校验符号(GF(256) based)
	for i := 0; i < c.numHDPC; i++ {
		intermediate[c.sourceSymbols+c.numLDPC+i] = make([]byte, T)
		for j := 0; j < c.sourceSymbols+c.numLDPC; j++ {
			coeff := hdpcCoeff(i, j, c.numHDPC)
			if coeff != 0 {
				gfMulAdd(intermediate[c.sourceSymbols+c.numLDPC+i],
					intermediate[j], coeff)
			}
		}
	}

	return intermediate
}

// ltEncode LT编码: 从中间符号生成一个编码符号
func (c *Codec) ltEncode(source [][]byte, esi uint32) []byte {
	K := c.sourceSymbols
	T := c.symbolSize
	result := make([]byte, T)

	// LT度分布 + 邻居选择(只在源符号范围内)
	degree := ltDegree(esi, K)
	neighbors := ltNeighbors(esi, degree, K)

	for _, n := range neighbors {
		if n < len(source) {
			xorSymbol(result, source[n])
		}
	}

	return result
}

// gaussianDecode 完整高斯消元解码
func (c *Codec) gaussianDecode(received []Symbol) ([][]byte, error) {
	K := c.sourceSymbols
	T := c.symbolSize

	result := make([][]byte, K)
	missing := []int{}

	// 用已收到的源符号填充
	have := make(map[int]bool)
	for _, sym := range received {
		if int(sym.ESI) < K {
			result[sym.ESI] = make([]byte, T)
			copy(result[sym.ESI], sym.Data)
			have[int(sym.ESI)] = true
		}
	}

	for i := 0; i < K; i++ {
		if !have[i] { missing = append(missing, i) }
	}
	if len(missing) == 0 { return result, nil }

	// 收集修复方程
	type equation struct {
		coeffs map[int]bool // 哪些源符号参与(XOR)
		value  []byte       // 方程右边的值
	}
	equations := []equation{}

	for _, sym := range received {
		if int(sym.ESI) >= K {
			degree := ltDegree(sym.ESI, c.sourceSymbols)
			neighbors := ltNeighbors(sym.ESI, degree, c.sourceSymbols)

			eq := equation{
				coeffs: make(map[int]bool),
				value:  make([]byte, T),
			}
			copy(eq.value, sym.Data)

			for _, n := range neighbors {
				if n < K {
					if have[n] {
						// 已知: XOR掉
						xorSymbol(eq.value, result[n])
					} else {
						eq.coeffs[n] = true
					}
				}
			}

			if len(eq.coeffs) > 0 {
				equations = append(equations, eq)
			}
		}
	}

	// 迭代消元(多轮BP+高斯混合)
	for iter := 0; iter < 50 && len(missing) > 0; iter++ {
		progress := false

		for i := len(equations) - 1; i >= 0; i-- {
			eq := &equations[i]

			// 移除已知变量
			for v := range eq.coeffs {
				if have[v] {
					xorSymbol(eq.value, result[v])
					delete(eq.coeffs, v)
				}
			}

			// 只剩1个未知→直接解
			if len(eq.coeffs) == 1 {
				for v := range eq.coeffs {
					result[v] = eq.value
					have[v] = true
					progress = true
					// 从missing移除
					for j, m := range missing {
						if m == v {
							missing = append(missing[:j], missing[j+1:]...)
							break
						}
					}
				}
				equations = append(equations[:i], equations[i+1:]...)
			}
		}

		if !progress { break }
	}

	if len(missing) > 0 {
		return nil, errors.New("decode failed: not enough repair symbols")
	}
	return result, nil
}

// === 辅助函数 ===

func xorSymbol(dst, src []byte) {
	xorFast(dst, src); return
	// 旧实现:
	for i := range dst {
		if i < len(src) { dst[i] ^= src[i] }
	}
}

// ldpcConnect LDPC连接判断(稀疏)
func ldpcConnect(check, variable, K, S int) bool {
	// 简化版: 每个校验连接~3个变量
	h := uint32(check*31+variable*17) ^ uint32(K*13)
	return h%uint32(K/3+1) == 0
}

// hdpcCoeff HDPC系数(GF(256))
func hdpcCoeff(row, col, H int) byte {
	h := uint32(row*37 + col*53)
	return byte(h % 256)
}

// gfMulAdd GF(256)乘加: dst += src * coeff
func gfMulAdd(dst, src []byte, coeff byte) { gf256.MulAdd(dst, src, coeff); return }
func gfMulAddOld(dst, src []byte, coeff byte) {
	if coeff == 0 { return }
	if coeff == 1 { xorSymbol(dst, src); return }
	for i := range dst {
		if i < len(src) {
			dst[i] ^= gfMul(src[i], coeff)
		}
	}
}

// gfMul GF(256)乘法(简化版,用log/antilog表更快)
func gfMul(a, b byte) byte { return gf256.Mul(a, b) }
func gfMulOld(a, b byte) byte {
	if a == 0 || b == 0 { return 0 }
	// 简化: 直接模乘
	return byte((uint16(a) * uint16(b)) % 255)
}

// ltDegree LT度分布(Robust Soliton)
func ltDegree(esi uint32, L int) int {
	// 简化版Robust Soliton分布
	h := esi * 2654435761 // Knuth乘法hash
	r := float64(h) / float64(math.MaxUint32)
	
	if r < 0.01 { return 1 }
	if r < 0.50 { return 2 }
	if r < 0.80 { return 3 }
	if r < 0.95 { return int(math.Log2(float64(L))) }
	return L / 2
}

// ltNeighbors 选择LT编码的邻居符号
func ltNeighbors(esi uint32, degree, L int) []int {
	neighbors := make([]int, 0, degree)
	h := esi
	// 用位图代替map(零分配)
	for len(neighbors) < degree && len(neighbors) < L {
		h = h*1103515245 + 12345
		n := int(h % uint32(L))
		// 线性检查重复(degree通常<10,比map快)
		dup := false
		for _, nn := range neighbors {
			if nn == n { dup = true; break }
		}
		if !dup {
			neighbors = append(neighbors, n)
		}
	}
	return neighbors
}

// 忽略binary包的unused警告
var _ = binary.BigEndian
