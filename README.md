# GoFEC

Pure Go FEC (Forward Error Correction) erasure code library. No CGO. AVX2 SIMD accelerated.

[English](#english)

## 特性

| 编码 | 包名 | 状态 | 性能 |
|------|------|------|------|
| RaptorQ | `raptorq` | ✅ | 43μs/32KB encode |
| LDPC | `ldpc` | ✅ | 18μs encode, 20μs decode |
| GF(256) | `internal/gf256` | ✅ | 11.5 GB/s (AVX2 VPSHUFB) |
| XOR | `internal/xor` | ✅ | 31.6 GB/s (AVX2) |

**为什么选GoFEC？**
- **纯Go** — 无CGO、无SWIG、无外部C/C++依赖
- **跨平台** — Windows / Linux / macOS / Android / iOS
- **SIMD加速** — AVX2 + SSE2 (amd64)，通用uint64 fallback (其他平台)
- **统一接口** — 所有编码实现统一Codec接口，可插拔切换

## 安装

```bash
go get github.com/nyarime/gofec
```

## 快速开始

### RaptorQ (喷泉码)

```go
import "github.com/nyarime/gofec/raptorq"

// 创建编解码器: 16个源符号, 每个128字节
codec := raptorq.New(16, 128)

// 编码: 源数据 + 8个修复符号
symbols := codec.Encode(data, 8)

// 丢失部分符号后解码
decoded, err := codec.Decode(receivedSymbols, len(data))
```

### LDPC

```go
import "github.com/nyarime/gofec/ldpc"

// 创建编解码器: 10数据块 + 4校验块, PEG矩阵密度0.3
codec := ldpc.New(10, 4, 0.3)

// 编码
encoded := codec.Encode(dataShards)

// 丢失部分块后恢复
err := codec.Decode(partialShards)
```

## 性能

### RaptorQ编码

| 数据大小 | 延迟 | 吞吐 |
|----------|------|------|
| 512B | 2.6μs | 197 MB/s |
| 2KB | 6.1μs | 328 MB/s |
| 8KB | 15μs | 533 MB/s |
| 32KB | 43μs | 744 MB/s |

### 丢失恢复

| 丢失块数 | 16源+N*3修复 | 状态 |
|----------|-------------|------|
| 1 | ✅ | 恢复成功 |
| 2 | ✅ | 恢复成功 |
| 3 | ✅ | 恢复成功 |
| 4 | ✅ | 恢复成功 |
| 5 | ✅ | 恢复成功 |
| 6 | ✅ | 恢复成功 |

### SIMD加速

| 操作 | 标量 | AVX2 | 提升 |
|------|------|------|------|
| XOR | 1 GB/s | 31.6 GB/s | 31x |
| GF(256) MulAdd | 773 MB/s | 11.5 GB/s | 15x |

## 架构

```
GoFEC/
├── codec.go              # 统一Codec接口
├── raptorq/              # RaptorQ喷泉码 (RFC 6330核心)
│   ├── raptorq.go        # 编解码器
│   └── xor.go            # AVX2 XOR
├── ldpc/                 # LDPC低密度校验码
│   ├── ldpc.go           # 编解码器
│   ├── peg.go            # PEG矩阵构造
│   └── xor.go            # AVX2 XOR
└── internal/
    ├── xor/              # AVX2/SSE2 XOR加速
    │   ├── xor_amd64.s   # 汇编
    │   └── xor_generic.go
    └── gf256/            # GF(256)有限域
        ├── gf.go         # Log/Exp查表
        ├── mulAdd_amd64.s # AVX2 VPSHUFB
        └── muladd_generic.go
```

## 被引用

- [NRUP](https://github.com/Nyarime/NRUP) — 高性能UDP传输协议的FEC层

## 致谢

- [klauspost/reedsolomon](https://github.com/klauspost/reedsolomon) — Go生态中最优秀的Reed-Solomon实现。GoFEC的AVX2 SIMD加速策略和GF(256) VPSHUFB查表设计深受其启发。感谢Klaus Post为Go社区做出的卓越贡献。
- [google/gofountain](https://github.com/google/gofountain) — 纯Go喷泉码参考实现。

## 许可证

Apache License 2.0

---

<a name="english"></a>
## English

Pure Go FEC (Forward Error Correction) erasure code library with AVX2 SIMD acceleration. No CGO required.

**Features:**
- RaptorQ fountain code (RFC 6330 core) — 43μs/32KB
- LDPC with PEG matrix — 18μs encode
- AVX2 VPSHUFB GF(256) — 11.5 GB/s
- AVX2 XOR — 31.6 GB/s
- Zero CGO, cross-platform

**GoFEC aims to be for erasure codes what klauspost/reedsolomon is for Reed-Solomon — the fastest pure Go implementation available.**

We believe in contributing back to the Go ecosystem. This library is our effort to provide high-performance FEC primitives that were previously only available in C/C++.

## Acknowledgments

- [klauspost/reedsolomon](https://github.com/klauspost/reedsolomon) — The gold standard Reed-Solomon implementation in Go. GoFEC's AVX2 SIMD acceleration and GF(256) VPSHUFB lookup table design are deeply inspired by this exceptional library. Thank you Klaus Post for your outstanding contributions to the Go ecosystem.
- [google/gofountain](https://github.com/google/gofountain) — Pure Go fountain code reference implementation.
