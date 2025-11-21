package main

import (
	"fmt"

	"github.com/junbin-yang/go-kitbox/pkg/bytesconv"
)

func main() {
	fmt.Println("=== BytesConv 示例 ===")
	fmt.Println()

	// 示例 1: 字符串转字节
	fmt.Println("1. 字符串转字节切片:")
	str := "Hello, 世界!"
	bytes := bytesconv.StringToBytes(str)
	fmt.Printf("   原字符串: %s\n", str)
	fmt.Printf("   转换结果: %v\n", bytes)
	fmt.Printf("   长度: %d\n", len(bytes))
	fmt.Println()

	// 示例 2: 字节转字符串
	fmt.Println("2. 字节切片转字符串:")
	data := []byte{72, 101, 108, 108, 111, 32, 71, 111, 33}
	text := bytesconv.BytesToString(data)
	fmt.Printf("   原字节: %v\n", data)
	fmt.Printf("   转换结果: %s\n", text)
	fmt.Println()

	// 示例 3: 往返转换
	fmt.Println("3. 往返转换:")
	original := "Go 语言"
	step1 := bytesconv.StringToBytes(original)
	step2 := bytesconv.BytesToString(step1)
	fmt.Printf("   原始: %s\n", original)
	fmt.Printf("   结果: %s\n", step2)
	fmt.Printf("   相等: %v\n", original == step2)
}
