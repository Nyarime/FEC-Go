# GoFEC

纯Go实现的FEC纠删码库集合。无CGO，跨平台。

## 支持的编码

| 编码 | 包名 | 状态 | 说明 |
|------|------|------|------|
| Reed-Solomon | `rs` | 📋 计划 | 经典纠删码(已在NRUP中使用klauspost版) |
| LDPC | `ldpc` | ✅ v1.0 | 低密度校验码，迭代BP解码 |
| RaptorQ | `raptorq` | 🔧 开发中 | RFC 6330喷泉码 |
| LT Code | `lt` | 📋 计划 | Luby Transform喷泉码 |

## 安装

```bash
go get github.com/nyarime/gofec
```

## 使用

```go
import "github.com/nyarime/gofec/ldpc"

// LDPC: 10数据块 + 4校验块, 密度0.3
codec := ldpc.New(10, 4, 0.3)
encoded := codec.Encode(dataShards)

// 丢失部分块后恢复
err := codec.Decode(partialShards)
```

## 设计原则

- **纯Go**: 无CGO，无SWIG，无外部C/C++依赖
- **跨平台**: Windows/Linux/macOS/Android/iOS
- **统一接口**: 所有编码实现统一Codec接口
- **可插拔**: 按需导入子包，不引入不需要的依赖

## 被引用

- [NRUP](https://github.com/Nyarime/NRUP) - UDP传输协议FEC层

## 许可证

Apache License 2.0
