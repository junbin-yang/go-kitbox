# Binpack - 二进制协议编解码库

高性能的二进制协议编解码库，通过结构体标签定义协议格式，支持代码生成和零反射开销。

## 特性

-   🚀 **高性能** - 支持反射缓存和预编译 codec
-   🏷️ **标签驱动** - 通过结构体 tag 定义协议格式
-   🔄 **字节序支持** - 支持大端序（BE）和小端序（LE）
-   📦 **零依赖** - 仅依赖 Go 标准库
-   ✅ **类型安全** - 编译时类型检查

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/binpack
```

## 快速开始

### 基础用法

```go
package main

import (
    "fmt"
    "github.com/junbin-yang/go-kitbox/pkg/binpack"
)

type GamePacket struct {
    Magic   uint32 `bin:"0:4:be"`      // 偏移 0，大小 4，大端序
    Type    uint8  `bin:"4:1"`         // 偏移 4，大小 1
    Length  uint16 `bin:"5:2:le"`      // 偏移 5，大小 2，小端序
    Payload [64]byte `bin:"7:64"`      // 偏移 7，大小 64
}

func main() {
    // 编码
    pkt := GamePacket{
        Magic:  0x12345678,
        Type:   1,
        Length: 64,
    }

    data, err := binpack.Marshal(&pkt)
    if err != nil {
        panic(err)
    }

    // 解码
    var decoded GamePacket
    err = binpack.Unmarshal(data, &decoded)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Magic: 0x%08X\n", decoded.Magic)
}
```

## Tag 语法

### 基础格式

**固定长度字段：**

```
bin:"offset:size:endian"
```

-   `offset`: 字节偏移量（从 0 开始）
-   `size`: 字节大小
-   `endian`: 字节序，`be`（大端）或 `le`（小端），默认 `be`

**变长字段：**

```
bin:"offset:var,len:LengthField"
```

-   `offset`: 字节偏移量
-   `var`: 标记为变长字段
-   `len:FieldName`: 指定长度来源字段名

**变长字符串：**

```
bin:"offset:var,len:LengthField,enc:encoding"
```

-   `enc`: 字符串编码方式（`utf8`/`ascii`/`hex`），默认 `utf8`

**位字段：**

```
bin:"offset:1,bits:bitIndex"
bin:"offset:1,bits:startBit-endBit"
```

-   `bits`: 位索引（0-7）或位范围（如 `3-4`）
-   必须指定 `size:1`（单字节）

**条件字段：**

```
bin:"offset:size:endian,if:FieldName==Value"
```

-   `if`: 条件表达式，仅支持相等比较（`==`）

### 示例

```go
type Protocol struct {
    // 基础类型
    U8  uint8   `bin:"0:1"`        // 1 字节
    U16 uint16  `bin:"1:2:be"`     // 2 字节，大端序
    U32 uint32  `bin:"3:4:le"`     // 4 字节，小端序
    U64 uint64  `bin:"7:8:be"`     // 8 字节，大端序

    // 有符号整数
    I8  int8    `bin:"15:1"`
    I16 int16   `bin:"16:2:le"`
    I32 int32   `bin:"18:4:le"`
    I64 int64   `bin:"22:8:le"`

    // 浮点数
    F32 float32 `bin:"30:4:be"`
    F64 float64 `bin:"34:8:be"`

    // 布尔值（1 字节）
    Flag bool   `bin:"42:1"`

    // 固定长度字节数组
    Data [16]byte `bin:"43:16"`

    // 变长字段（需要指定长度来源字段）
    Length  uint16 `bin:"59:2:le"`
    Payload []byte `bin:"61:var,len:Length"`

    // 变长字符串
    NameLen uint8  `bin:"100:1"`
    Name    string `bin:"101:var,len:NameLen"`

    // 固定长度字符串
    Title string `bin:"200:32"`

    // 跳过字段（不参与编解码）
    Internal string `bin:"-"`
}
```

## 支持的类型

| 类型                                  | 说明                                   |
| ------------------------------------- | -------------------------------------- |
| `uint8`, `uint16`, `uint32`, `uint64` | 无符号整数                             |
| `int8`, `int16`, `int32`, `int64`     | 有符号整数                             |
| `float32`, `float64`                  | 浮点数                                 |
| `bool`                                | 布尔值（1 字节）                       |
| `[N]byte`                             | 固定长度字节数组                       |
| `[]byte`                              | 变长字节切片（需指定 `len:FieldName`） |
| `string`                              | 字符串（固定长度或变长）               |
| `[]T` (基础类型)                      | 数组字段（需指定 `repeat,size`）       |
| `[]Struct`                            | 结构体数组（需指定 `repeat`）          |

## 变长字段

变长字段需要通过 `len:FieldName` 选项指定长度来源字段：

```go
type Packet struct {
    Magic   uint32 `bin:"0:4:be"`
    Length  uint16 `bin:"4:2:le"`      // 长度字段
    Payload []byte `bin:"6:var,len:Length"` // 变长字段，长度由 Length 指定
}

// 使用示例
pkt := Packet{
    Magic:   0x12345678,
    Length:  13,
    Payload: []byte("Hello, World!"),
}

data, _ := binpack.Marshal(&pkt)
// data = [0x12 0x34 0x56 0x78 0x0D 0x00 'H' 'e' 'l' 'l' 'o' ',' ' ' 'W' 'o' 'r' 'l' 'd' '!']
```

**注意事项**：

-   长度字段必须在变长字段之前定义
-   长度字段必须是无符号整数类型（uint8/16/32/64）
-   编码时会自动根据实际数据长度写入

## 字符串编码

支持通过 `enc:` 选项指定字符串编码方式：

```go
type Packet struct {
    Magic uint32 `bin:"0:4:be"`

    // UTF-8 编码（默认）
    Name string `bin:"4:32"`

    // ASCII 编码
    Title string `bin:"36:64,enc:ascii"`

    // Hex 编码（十六进制）
    Hash string `bin:"100:64,enc:hex"`
}

// 使用示例
pkt := Packet{
    Magic: 0x12345678,
    Name:  "Alice",
    Title: "Engineer",
    Hash:  "Test!",  // 编码为 "5465737421"
}
```

**支持的编码**：

-   `utf8` 或留空：UTF-8 编码（默认）
-   `ascii`：ASCII 编码
-   `hex`：十六进制编码（将字符串转为十六进制表示）

**Hex 编码说明**：

-   编码：每个字节转为 2 个十六进制字符（如 `"A"` → `"41"`）
-   解码：每 2 个十六进制字符转为 1 个字节
-   用途：适合存储哈希值、二进制数据的可读表示

## 位字段

支持通过 `bits:` 选项解析单个字节内的位字段：

```go
type Flags struct {
    Status uint8 `bin:"0:1"`

    // 位字段（共享同一字节）
    Enable  uint8 `bin:"1:1,bits:0"`     // 位 0
    Mode    uint8 `bin:"1:1,bits:1-2"`   // 位 1-2（2 位）
    Level   uint8 `bin:"1:1,bits:3-4"`   // 位 3-4（2 位）
    Debug   uint8 `bin:"1:1,bits:5"`     // 位 5
}

// 使用示例
flags := Flags{
    Status: 0xFF,
    Enable: 1,    // 0b1
    Mode:   2,    // 0b10
    Level:  3,    // 0b11
    Debug:  1,    // 0b1
}
// 字节1 = 0b00111101 = 0x3D
```

**注意事项**：

-   位字段必须指定 `size:1`（单字节）
-   位索引范围：0-7
-   支持单个位（`bits:5`）或位范围（`bits:3-4`）
-   多个位字段可以共享同一字节

## 条件字段

支持通过 `if:` 选项根据其他字段的值决定是否编解码：

```go
type Packet struct {
    Type   uint8  `bin:"0:1"`
    Length uint16 `bin:"1:2:le"`

    // 条件字段
    Data1  uint32 `bin:"3:4:be,if:Type==1"` // 仅当 Type==1 时编码
    Data2  uint32 `bin:"3:4:be,if:Type==2"` // 仅当 Type==2 时编码
}

// 使用示例
pkt1 := Packet{
    Type:  1,
    Data1: 0x12345678, // 会被编码
    Data2: 0xABCDEF00, // 不会被编码
}

pkt2 := Packet{
    Type:  2,
    Data1: 0x12345678, // 不会被编码
    Data2: 0xABCDEF00, // 会被编码
}
```

**注意事项**：

-   条件格式：`if:FieldName==Value`
-   条件字段必须在被引用字段之后定义
-   仅支持无符号整数类型的相等比较
-   用途：协议版本兼容、可选字段

## 数组字段

支持通过 `repeat` 选项解析重复的字段数组：

### 基础类型数组

```go
type ModbusPacket struct {
    RegisterCount uint8    `bin:"0:1"`
    Registers     []uint16 `bin:"1:var,len:RegisterCount,repeat,size:2:be"`
}

// 使用示例
pkt := ModbusPacket{
    RegisterCount: 3,
    Registers:     []uint16{100, 200, 300},
}
// 编码: [0x03 0x00 0x64 0x00 0xC8 0x01 0x2C]
```

**语法**: `bin:"offset:var,len:LengthField,repeat,size:ElementSize:endian"`

-   `repeat`: 标记为数组字段
-   `size:N`: 每个元素的字节大小（基础类型必需）
-   `endian`: 字节序（可选，be/le）

### 结构体数组

```go
type CoAPOption struct {
    Delta  uint8  `bin:"0:1"`
    Length uint8  `bin:"1:1"`
    Value  []byte `bin:"2:var,len:Length"`
}

type CoAPPacket struct {
    OptionCount uint8        `bin:"0:1"`
    Options     []CoAPOption `bin:"1:var,len:OptionCount,repeat"`
}

// 使用示例
pkt := CoAPPacket{
    OptionCount: 2,
    Options: []CoAPOption{
        {Delta: 11, Length: 4, Value: []byte("host")},
        {Delta: 15, Length: 4, Value: []byte("path")},
    },
}
```

**语法**: `bin:"offset:var,len:LengthField,repeat"`

-   自动计算每个结构体元素的大小
-   支持嵌套的变长字段

### 固定长度数组

```go
type SensorData struct {
    Timestamp uint32     `bin:"0:4:be"`
    Readings  [10]uint16 `bin:"4:20:be,repeat,size:2"`
}
```

**语法**: `bin:"offset:totalSize:endian,repeat,size:ElementSize"`

## 高性能用法

### 预编译 Codec

```go
import "reflect"

type Packet struct {
    Magic uint32 `bin:"0:4:be"`
    Type  uint8  `bin:"4:1"`
}

// 预编译 codec（只需执行一次）
var packetCodec = binpack.MustCompile(reflect.TypeOf(Packet{}))

func encode(pkt *Packet) ([]byte, error) {
    // 复用 codec，避免反射开销
    return packetCodec.Encode(pkt)
}

func decode(data []byte) (*Packet, error) {
    var pkt Packet
    err := packetCodec.Decode(data, &pkt)
    return &pkt, err
}
```

### 使用预分配 Buffer

```go
// 预分配 buffer
buf := make([]byte, 1024)

// 编码到 buffer
n, err := binpack.MarshalTo(buf, &pkt)
if err != nil {
    panic(err)
}

// 使用 buf[:n]
```

### Buffer 池（零拷贝）

```go
// 创建 buffer 池
pool := binpack.NewBufferPool(1024)

// 零拷贝编码（返回池中的 buffer）
data, err := binpack.MarshalWithPool(pool, &pkt)
if err != nil {
    panic(err)
}

// 使用完毕后归还 buffer
defer pool.Put(data)

// 或使用带复制的版本（自动归还）
data, err := binpack.MarshalWithPoolCopy(pool, &pkt)
```

### 代码生成器（消除反射开销）

```go
import (
    "reflect"
    "github.com/junbin-yang/go-kitbox/pkg/binpack/generator"
)

type Packet struct {
    Magic  uint32 `bin:"0:4:be"`
    Type   uint8  `bin:"4:1"`
    Length uint16 `bin:"5:2:le"`
}

// 生成静态编解码代码
code, err := generator.Generate(reflect.TypeOf(Packet{}), "mypackage")
if err != nil {
    panic(err)
}

// 将生成的代码写入文件
os.WriteFile("packet_codec.go", code, 0644)

// 生成的代码示例：
// func MarshalPacket(v *Packet) ([]byte, error) {
//     buf := make([]byte, 7)
//     binary.BigEndian.PutUint32(buf[0:], v.Magic)
//     buf[4] = v.Type
//     binary.LittleEndian.PutUint16(buf[5:], v.Length)
//     return buf, nil
// }
```

**优势**：

-   零反射开销：生成的代码直接操作字段，无需运行时反射
-   类型安全：编译时检查，避免运行时错误
-   性能提升：比反射模式快 2-3 倍
-   可读性强：生成的代码清晰易懂，便于调试

## 与网络库集成

### 与 netconn 集成

```go
import "github.com/junbin-yang/go-kitbox/pkg/netconn"

// 读取数据包
conn, _ := netconn.Dial("tcp", "localhost:8080")

// 方式1: 先读取固定头部，再读取负载
type Header struct {
    Magic  uint32 `bin:"0:4:be"`
    Length uint16 `bin:"4:2:le"`
}

headerData := make([]byte, 6)
conn.Read(headerData)

var header Header
binpack.Unmarshal(headerData, &header)

payloadData := make([]byte, header.Length)
conn.Read(payloadData)

// 方式2: 一次性读取并解析
data := make([]byte, 1024)
n, _ := conn.Read(data)

var pkt GamePacket
binpack.Unmarshal(data[:n], &pkt)

// 发送数据包
data, _ := binpack.Marshal(&pkt)
conn.Write(data)
```

### 与标准 net 包集成

```go
import "net"

conn, _ := net.Dial("tcp", "localhost:8080")

// 读取
buf := make([]byte, 1024)
n, _ := conn.Read(buf)

var pkt GamePacket
binpack.Unmarshal(buf[:n], &pkt)

// 写入
data, _ := binpack.Marshal(&pkt)
conn.Write(data)
```

## 性能

### 基础性能测试（复杂结构体）

测试场景：包含多种字段类型的复杂结构体（80 字节）

```
BenchmarkMarshal-20                 7864705   172.6 ns/op    80 B/op    1 allocs/op
BenchmarkUnmarshal-20               7551505   158.0 ns/op    80 B/op    1 allocs/op
BenchmarkMarshalWithCodec-20        8868666   124.0 ns/op    80 B/op    1 allocs/op
BenchmarkMarshalWithPool-20         7743098   182.4 ns/op    24 B/op    1 allocs/op
BenchmarkMarshalWithPoolCopy-20     5997828   235.1 ns/op   104 B/op    2 allocs/op
```

**性能对比**：

-   反射模式：~173 ns/op, 80 B/op, 1 allocs/op
-   预编译模式：~124 ns/op, 80 B/op, 1 allocs/op（提升 28%）
-   Buffer 池（零拷贝）：~182 ns/op, 24 B/op, 1 allocs/op（内存分配减少 70%）
-   Buffer 池（带复制）：~235 ns/op, 104 B/op, 2 allocs/op

### 代码生成性能测试（简单结构体）

测试场景：简单协议包结构体（7 字节，3 个字段）

| 方式       | 性能        | 内存分配            | 说明       |
| ---------- | ----------- | ------------------- | ---------- |
| 反射模式   | ~43 ns/op   | 8 B/op, 1 allocs/op | 运行时反射 |
| 预编译模式 | ~28 ns/op   | 8 B/op, 1 allocs/op | 缓存 codec |
| 代码生成   | ~0.22 ns/op | 0 B/op, 0 allocs/op | 零反射开销 |

代码生成模式比反射模式快约 **190 倍**，且零内存分配。

## 最佳实践

### 1. 协议设计建议

```go
// ✅ 推荐：头部和负载分离
type Header struct {
    Magic  uint32 `bin:"0:4:be"`
    Type   uint8  `bin:"4:1"`
    Length uint16 `bin:"5:2:le"`
}

type Payload struct {
    Data []byte
}

// ✅ 推荐：合并为一个结构体
type Packet struct {
    Magic   uint32   `bin:"0:4:be"`
    Type    uint8    `bin:"4:1"`
    Length  uint16   `bin:"5:2:le"`
    Payload [256]byte `bin:"7:256"`
}
```

### 2. 性能优化

```go
// ✅ 推荐：预编译 codec
var codec = binpack.MustCompile(reflect.TypeOf(Packet{}))

// ✅ 推荐：使用 buffer 池
var pool = binpack.NewBufferPool(1024)

func encode(pkt *Packet) []byte {
    // 零拷贝版本（需要手动归还）
    data, _ := binpack.MarshalWithPool(pool, pkt)
    defer pool.Put(data)
    return data
}

func encodeCopy(pkt *Packet) []byte {
    // 带复制版本（自动归还）
    data, _ := binpack.MarshalWithPoolCopy(pool, pkt)
    return data
}
```

### 3. 错误处理

binpack 提供详细的错误信息,包含字段名、偏移位置、期望长度等,便于快速定位问题。

```go
// ✅ 推荐：检查错误
var pkt GamePacket
err := binpack.Unmarshal(data, &pkt)
if err != nil {
    // 详细错误示例:
    // field "MessageID" (uint16) at offset 16: expected 2 bytes, got 1 bytes: data too short
    // field "Enable" (uint8) at offset 1 bit 0: expected 1 bits, got 0 bits: bit not set
    log.Printf("unmarshal failed: %v", err)
    return err
}

// ✅ 推荐：使用类型断言获取详细错误信息
if decErr, ok := err.(*binpack.DecodeError); ok {
    log.Printf("解码失败: 字段=%s, 类型=%s, 偏移=%d, 期望=%d字节, 实际=%d字节",
        decErr.FieldName, decErr.FieldType, decErr.Offset,
        decErr.ExpectedSize, decErr.ActualSize)
}

// ✅ 推荐：验证数据长度
if len(data) < expectedSize {
    return fmt.Errorf("data too short: expected %d, got %d", expectedSize, len(data))
}
```

**错误类型:**

- `DecodeError`: 解码错误,包含字段名、类型、偏移、期望/实际长度等详细信息
- `EncodeError`: 编码错误,包含字段名、类型和错误描述

## CLI 工具

### binpack-gen 代码生成器

`binpack-gen` 是一个命令行工具，可以为结构体生成静态编解码代码，消除反射开销。

#### 安装

```bash
go install github.com/junbin-yang/go-kitbox/pkg/binpack/generator/cmd/binpack-gen@latest
```

或从源码构建：

```bash
go build -o binpack-gen pkg/binpack/generator/cmd/binpack-gen.go
```

#### 使用方法

```bash
binpack-gen -pkg <package> -type <struct> [-output <file>]
```

参数说明：

-   `-pkg`: 包路径（如 `./mypackage`）
-   `-type`: 结构体类型名
-   `-output`: 输出文件路径（可选，默认输出到标准输出）

#### 示例

```bash
# 为 Packet 结构体生成代码
binpack-gen -pkg ./internal/binpack/testdata -type Packet -output packet_gen.go

# 输出到标准输出
binpack-gen -pkg ./mypackage -type GamePacket
```

#### 生成的代码

生成的代码包含两个函数：

```go
// Marshal<TypeName> 编码结构体
func MarshalPacket(v *Packet) ([]byte, error) {
    buf := make([]byte, 7)
    binary.BigEndian.PutUint32(buf[0:], v.Magic)
    buf[4] = v.Type
    binary.LittleEndian.PutUint16(buf[5:], v.Length)
    return buf, nil
}

// Unmarshal<TypeName> 解码结构体
func UnmarshalPacket(data []byte, v *Packet) error {
    v.Magic = binary.BigEndian.Uint32(data[0:])
    v.Type = data[4]
    v.Length = binary.LittleEndian.Uint16(data[5:])
    return nil
}
```

## 许可证

MIT License - 详见 [LICENSE](../../LICENSE)
