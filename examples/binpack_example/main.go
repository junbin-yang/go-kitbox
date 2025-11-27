package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/junbin-yang/go-kitbox/pkg/binpack"
	"github.com/junbin-yang/go-kitbox/pkg/binpack/generator"
)

type GamePacket struct {
	Magic  uint32 `bin:"0:4:be"`
	Type   uint8  `bin:"4:1"`
	Length uint16 `bin:"5:2:le"`
	Data   uint32 `bin:"7:4:be"`
}

func main() {
	fmt.Println("=== binpack 示例 ===\n")

	// 1. 基础编解码
	fmt.Println("1. 基础编解码:")
	pkt := GamePacket{
		Magic:  0x12345678,
		Type:   1,
		Length: 100,
		Data:   0xABCDEF00,
	}

	data, err := binpack.Marshal(&pkt)
	if err != nil {
		panic(err)
	}
	fmt.Printf("   编码结果: %x\n", data)

	var decoded GamePacket
	err = binpack.Unmarshal(data, &decoded)
	if err != nil {
		panic(err)
	}
	fmt.Printf("   解码结果: Magic=0x%08X, Type=%d, Length=%d, Data=0x%08X\n\n",
		decoded.Magic, decoded.Type, decoded.Length, decoded.Data)

	// 2. 使用 Buffer 池
	fmt.Println("2. 使用 Buffer 池:")
	pool := binpack.NewBufferPool(1024)
	poolData, err := binpack.MarshalWithPoolCopy(pool, &pkt)
	if err != nil {
		panic(err)
	}
	fmt.Printf("   编码结果: %x\n\n", poolData)

	// 3. 代码生成器
	fmt.Println("3. 代码生成器:")
	code, err := generator.Generate(reflect.TypeOf(GamePacket{}), "mypackage")
	if err != nil {
		panic(err)
	}

	// 写入文件
	filename := "packet_gen.go"
	if err := os.WriteFile(filename, code, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("   生成的代码已写入: %s\n", filename)
	fmt.Printf("   代码预览:\n%s\n", string(code))
}
