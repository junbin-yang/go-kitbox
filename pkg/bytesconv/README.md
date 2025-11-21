# BytesConv - 字节转换

高性能的零拷贝的字符串与字节切片转换工具。

## 特性

-   零内存分配
-   零拷贝转换
-   高性能（比标准转换快数倍）
-   支持 Unicode 字符串

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/bytesconv
```

## 使用方法

### 字符串转字节切片

```go
import "github.com/junbin-yang/go-kitbox/pkg/bytesconv"

s := "hello world"
b := bytesconv.StringToBytes(s)
// b 是 []byte 类型，零拷贝
```

### 字节切片转字符串

```go
b := []byte("hello world")
s := bytesconv.BytesToString(b)
// s 是 string 类型，零拷贝
```

## 完整示例

```go
package main

import (
    "fmt"
    "github.com/junbin-yang/go-kitbox/pkg/bytesconv"
)

func main() {
    // 字符串转字节
    str := "Hello, 世界!"
    bytes := bytesconv.StringToBytes(str)
    fmt.Printf("String to bytes: %v\n", bytes)

    // 字节转字符串
    data := []byte{72, 101, 108, 108, 111}
    text := bytesconv.BytesToString(data)
    fmt.Printf("Bytes to string: %s\n", text)
}
```

## 性能对比

```
BenchmarkStringToBytes-8              1000000000    0.25 ns/op    0 B/op    0 allocs/op
BenchmarkStandardStringToBytes-8      50000000      30.5 ns/op    32 B/op   1 allocs/op

BenchmarkBytesToString-8              1000000000    0.25 ns/op    0 B/op    0 allocs/op
BenchmarkStandardBytesToString-8      50000000      28.2 ns/op    32 B/op   1 allocs/op
```

零拷贝转换比标准转换快约 **100 倍**，且无内存分配。

## 注意事项

⚠️ **重要警告**

1. **不可修改原数据**：转换后不要修改原始数据，否则可能导致未定义行为
2. **生命周期**：确保原始数据在使用期间保持有效
3. **只读使用**：转换结果应仅用于只读场景

### 安全示例

```go
// ✅ 安全：只读使用
s := "hello"
b := bytesconv.StringToBytes(s)
fmt.Println(len(b)) // OK

// ❌ 危险：修改数据
b[0] = 'H' // 可能导致 panic 或未定义行为
```

## API 参考

### StringToBytes

```go
func StringToBytes(s string) []byte
```

将字符串转换为字节切片，零拷贝。

**参数：**

-   `s`：要转换的字符串

**返回：**

-   `[]byte`：字节切片（与原字符串共享底层数据）

### BytesToString

```go
func BytesToString(b []byte) string
```

将字节切片转换为字符串，零拷贝。

**参数：**

-   `b`：要转换的字节切片

**返回：**

-   `string`：字符串（与原字节切片共享底层数据）

## 使用场景

适用于：

-   高性能网络编程
-   大量字符串/字节转换场景
-   性能敏感的数据处理
-   只读数据转换

不适用于：

-   需要修改转换后数据的场景
-   数据生命周期不明确的场景

## 许可证

MIT License
