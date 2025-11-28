package binpack

import (
	"fmt"
	"reflect"
)

// Marshal 将结构体编码为字节流
func Marshal(v interface{}) ([]byte, error) {
	if err := ValidateStruct(v); err != nil {
		return nil, err
	}
	typ := reflect.TypeOf(v)
	codec, err := CompileCodec(typ)
	if err != nil {
		return nil, err
	}
	return codec.Encode(v)
}

// Unmarshal 将字节流解码为结构体
func Unmarshal(data []byte, v interface{}) error {
	if err := ValidateStruct(v); err != nil {
		return err
	}
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}

	codec, err := CompileCodec(typ)
	if err != nil {
		return err
	}
	return codec.Decode(data, v)
}

// MarshalTo 将结构体编码到指定 buffer
func MarshalTo(buf []byte, v interface{}) (int, error) {
	if err := ValidateStruct(v); err != nil {
		return 0, err
	}
	typ := reflect.TypeOf(v)
	codec, err := CompileCodec(typ)
	if err != nil {
		return 0, err
	}
	return codec.EncodeTo(buf, v)
}

// MarshalWithPool 使用 buffer 池编码结构体（零拷贝）
// 注意：返回的 []byte 引用池中的 buffer，使用完毕后需调用 pool.Put() 归还
func MarshalWithPool(pool *BufferPool, v interface{}) ([]byte, error) {
	buf := pool.Get()
	n, err := MarshalTo(buf, v)
	if err != nil {
		pool.Put(buf)
		return nil, err
	}
	return buf[:n], nil
}

// MarshalWithPoolCopy 使用 buffer 池编码结构体（带复制）
func MarshalWithPoolCopy(pool *BufferPool, v interface{}) ([]byte, error) {
	buf := pool.Get()
	defer pool.Put(buf)
	n, err := MarshalTo(buf, v)
	if err != nil {
		return nil, err
	}
	result := make([]byte, n)
	copy(result, buf[:n])
	return result, nil
}
