package binpack

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Codec 编解码器接口
type Codec interface {
	Encode(v interface{}) ([]byte, error)
	Decode(data []byte, v interface{}) error
	EncodeTo(buf []byte, v interface{}) (int, error)
}

// fieldCodec 处理单个字段的编解码
type fieldCodec struct {
	name        string // 字段名
	typeName    string // 字段类型名
	index       int
	offset      int
	size        int
	byteOrder   binary.ByteOrder
	lenField    string // 变长字段的长度来源字段名
	lenIndex    int    // 长度字段的索引
	isVariable  bool   // 是否为变长字段
	encoding    string // 字符串编码方式
	bitStart    int    // 位字段起始位（0-7）
	bitEnd      int    // 位字段结束位（0-7）
	isBitField  bool   // 是否为位字段
	condField   string // 条件字段名
	condIndex   int    // 条件字段索引
	condValue   uint64 // 条件值
	conditional bool   // 是否为条件字段
	isRepeat    bool   // 是否为数组字段
	elementSize int    // 每个元素的字节大小
	encoder     func(buf []byte, v reflect.Value) error
	decoder     func(data []byte, v reflect.Value) error
}

// encodeUint8 编码 uint8
func encodeUint8(buf []byte, v reflect.Value) error {
	buf[0] = uint8(v.Uint())
	return nil
}

// decodeUint8 解码 uint8
func decodeUint8(data []byte, v reflect.Value) error {
	v.SetUint(uint64(data[0]))
	return nil
}

// encodeUint16 编码 uint16
func encodeUint16(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint16(buf, uint16(v.Uint()))
		return nil
	}
}

// decodeUint16 解码 uint16
func decodeUint16(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetUint(uint64(order.Uint16(data)))
		return nil
	}
}

// encodeUint32 编码 uint32
func encodeUint32(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint32(buf, uint32(v.Uint()))
		return nil
	}
}

// decodeUint32 解码 uint32
func decodeUint32(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetUint(uint64(order.Uint32(data)))
		return nil
	}
}

// encodeUint64 编码 uint64
func encodeUint64(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint64(buf, v.Uint())
		return nil
	}
}

// decodeUint64 解码 uint64
func decodeUint64(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetUint(order.Uint64(data))
		return nil
	}
}

// encodeInt8 编码 int8
func encodeInt8(buf []byte, v reflect.Value) error {
	buf[0] = uint8(v.Int())
	return nil
}

// decodeInt8 解码 int8
func decodeInt8(data []byte, v reflect.Value) error {
	v.SetInt(int64(int8(data[0])))
	return nil
}

// encodeInt16 编码 int16
func encodeInt16(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint16(buf, uint16(v.Int()))
		return nil
	}
}

// decodeInt16 解码 int16
func decodeInt16(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetInt(int64(int16(order.Uint16(data))))
		return nil
	}
}

// encodeInt32 编码 int32
func encodeInt32(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint32(buf, uint32(v.Int()))
		return nil
	}
}

// decodeInt32 解码 int32
func decodeInt32(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetInt(int64(int32(order.Uint32(data))))
		return nil
	}
}

// encodeInt64 编码 int64
func encodeInt64(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint64(buf, uint64(v.Int()))
		return nil
	}
}

// decodeInt64 解码 int64
func decodeInt64(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetInt(int64(order.Uint64(data)))
		return nil
	}
}

// encodeFloat32 编码 float32
func encodeFloat32(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint32(buf, math.Float32bits(float32(v.Float())))
		return nil
	}
}

// decodeFloat32 解码 float32
func decodeFloat32(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetFloat(float64(math.Float32frombits(order.Uint32(data))))
		return nil
	}
}

// encodeFloat64 编码 float64
func encodeFloat64(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		order.PutUint64(buf, math.Float64bits(v.Float()))
		return nil
	}
}

// decodeFloat64 解码 float64
func decodeFloat64(order binary.ByteOrder) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetFloat(math.Float64frombits(order.Uint64(data)))
		return nil
	}
}

// encodeBool 编码 bool
func encodeBool(buf []byte, v reflect.Value) error {
	if v.Bool() {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	return nil
}

// decodeBool 解码 bool
func decodeBool(data []byte, v reflect.Value) error {
	v.SetBool(data[0] != 0)
	return nil
}

// encodeByteArray 编码 [N]byte
func encodeByteArray(size int) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		reflect.Copy(reflect.ValueOf(buf[:size]), v)
		return nil
	}
}

// decodeByteArray 解码 [N]byte
func decodeByteArray(size int) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		reflect.Copy(v, reflect.ValueOf(data[:size]))
		return nil
	}
}

// encodeByteSlice 编码 []byte
func encodeByteSlice(buf []byte, v reflect.Value) error {
	copy(buf, v.Bytes())
	return nil
}


// encodeString 编码 string（UTF-8，默认）- 零拷贝优化
func encodeString(buf []byte, v reflect.Value) error {
	s := v.String()
	copy(buf, s)
	return nil
}

// decodeString 解码 string（UTF-8，默认）
func decodeString(size int) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		v.SetString(string(data[:size]))
		return nil
	}
}

// encodeStringWithEncoding 使用指定编码编码 string
func encodeStringWithEncoding(encoding string) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		s := v.String()
		switch encoding {
		case "hex":
			// Hex 编码：将字符串转为十六进制
			src := []byte(s)
			for i := 0; i < len(src) && i*2 < len(buf); i++ {
				buf[i*2] = "0123456789abcdef"[src[i]>>4]
				buf[i*2+1] = "0123456789abcdef"[src[i]&0x0f]
			}
		default: // utf8, ascii
			copy(buf, s)
		}
		return nil
	}
}

// decodeStringWithEncoding 使用指定编码解码 string
func decodeStringWithEncoding(size int, encoding string) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		switch encoding {
		case "hex":
			// Hex 解码：将十六进制转为字符串
			dst := make([]byte, size/2)
			for i := 0; i < len(dst) && i*2+1 < len(data); i++ {
				hi := hexCharToNibble(data[i*2])
				lo := hexCharToNibble(data[i*2+1])
				dst[i] = (hi << 4) | lo
			}
			v.SetString(string(dst))
		default: // utf8, ascii
			v.SetString(string(data[:size]))
		}
		return nil
	}
}

// hexCharToNibble 将十六进制字符转为数字
func hexCharToNibble(c byte) byte {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	return 0
}

// encodeBitField 编码位字段
func encodeBitField(bitStart, bitEnd int) func([]byte, reflect.Value) error {
	return func(buf []byte, v reflect.Value) error {
		val := uint8(v.Uint())
		mask := uint8((1 << (bitEnd - bitStart + 1)) - 1)
		val &= mask
		buf[0] = (buf[0] & ^(mask << bitStart)) | (val << bitStart)
		return nil
	}
}

// decodeBitField 解码位字段
func decodeBitField(bitStart, bitEnd int) func([]byte, reflect.Value) error {
	return func(data []byte, v reflect.Value) error {
		mask := uint8((1 << (bitEnd - bitStart + 1)) - 1)
		val := (data[0] >> bitStart) & mask
		v.SetUint(uint64(val))
		return nil
	}
}

// getByteOrder 根据字节序字符串返回 binary.ByteOrder
func getByteOrder(endian string) binary.ByteOrder {
	if endian == "le" {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

// buildFieldCodec 为字段构建编解码器
func buildFieldCodec(field reflect.StructField, tag *tagInfo) (*fieldCodec, error) {
	if tag.Skip {
		return nil, nil
	}

	fc := &fieldCodec{
		name:        field.Name,
		typeName:    field.Type.String(),
		offset:      tag.Offset,
		size:        tag.Size,
		byteOrder:   getByteOrder(tag.ByteOrder),
		isRepeat:    tag.IsRepeat,
		elementSize: tag.ElementSize,
	}

	// 处理条件字段
	if tag.Condition != "" {
		parts := strings.Split(tag.Condition, "==")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid condition format, expected Field==Value")
		}
		fc.conditional = true
		fc.condField = strings.TrimSpace(parts[0])
		condVal, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 0, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid condition value: %w", err)
		}
		fc.condValue = condVal
	}

	// 处理位字段
	if tag.Bits != "" {
		if tag.Size != 1 {
			return nil, fmt.Errorf("bit fields must have size 1")
		}
		var bitStart, bitEnd int
		if strings.Contains(tag.Bits, "-") {
			parts := strings.Split(tag.Bits, "-")
			bitStart, _ = strconv.Atoi(parts[0])
			bitEnd, _ = strconv.Atoi(parts[1])
		} else {
			bitStart, _ = strconv.Atoi(tag.Bits)
			bitEnd = bitStart
		}
		fc.isBitField = true
		fc.bitStart = bitStart
		fc.bitEnd = bitEnd
		fc.encoder = encodeBitField(bitStart, bitEnd)
		fc.decoder = decodeBitField(bitStart, bitEnd)
		return fc, nil
	}

	kind := field.Type.Kind()

	switch kind {
	case reflect.Uint8:
		fc.encoder = encodeUint8
		fc.decoder = decodeUint8
	case reflect.Uint16:
		fc.encoder = encodeUint16(fc.byteOrder)
		fc.decoder = decodeUint16(fc.byteOrder)
	case reflect.Uint32:
		fc.encoder = encodeUint32(fc.byteOrder)
		fc.decoder = decodeUint32(fc.byteOrder)
	case reflect.Uint64:
		fc.encoder = encodeUint64(fc.byteOrder)
		fc.decoder = decodeUint64(fc.byteOrder)
	case reflect.Int8:
		fc.encoder = encodeInt8
		fc.decoder = decodeInt8
	case reflect.Int16:
		fc.encoder = encodeInt16(fc.byteOrder)
		fc.decoder = decodeInt16(fc.byteOrder)
	case reflect.Int32:
		fc.encoder = encodeInt32(fc.byteOrder)
		fc.decoder = decodeInt32(fc.byteOrder)
	case reflect.Int64:
		fc.encoder = encodeInt64(fc.byteOrder)
		fc.decoder = decodeInt64(fc.byteOrder)
	case reflect.Float32:
		fc.encoder = encodeFloat32(fc.byteOrder)
		fc.decoder = decodeFloat32(fc.byteOrder)
	case reflect.Float64:
		fc.encoder = encodeFloat64(fc.byteOrder)
		fc.decoder = decodeFloat64(fc.byteOrder)
	case reflect.Bool:
		fc.encoder = encodeBool
		fc.decoder = decodeBool
	case reflect.Array:
		if field.Type.Elem().Kind() == reflect.Uint8 {
			fc.encoder = encodeByteArray(tag.Size)
			fc.decoder = decodeByteArray(tag.Size)
		} else {
			return nil, fmt.Errorf("unsupported array type: %v", field.Type)
		}
	case reflect.Slice:
		if field.Type.Elem().Kind() == reflect.Uint8 {
			if tag.Size == -1 {
				// 变长字段
				if tag.LenField == "" {
					return nil, fmt.Errorf("variable length field requires len option")
				}
				fc.isVariable = true
				fc.lenField = tag.LenField
				fc.encoder = encodeByteSlice
				// decoder 将在 reflect_codec 中动态设置
			} else {
				return nil, fmt.Errorf("[]byte must be variable length, use [N]byte for fixed length")
			}
		} else if tag.IsRepeat {
			// 数组字段
			if tag.Size == -1 {
				if tag.LenField == "" {
					return nil, fmt.Errorf("array field requires len option")
				}
				fc.isVariable = true
				fc.lenField = tag.LenField
			}
			// encoder/decoder 将在 reflect_codec 中动态设置
		} else {
			return nil, fmt.Errorf("unsupported slice type: %v, use repeat option for arrays", field.Type)
		}
	case reflect.String:
		if tag.Size == -1 {
			// 变长字符串
			if tag.LenField == "" {
				return nil, fmt.Errorf("variable length string requires len option")
			}
			fc.isVariable = true
			fc.lenField = tag.LenField
			fc.encoding = tag.Encoding
			if tag.Encoding != "" && tag.Encoding != "utf8" {
				fc.encoder = encodeStringWithEncoding(tag.Encoding)
			} else {
				fc.encoder = encodeString
			}
			// decoder 将在 reflect_codec 中动态设置
		} else {
			// 固定长度字符串
			fc.encoding = tag.Encoding
			if tag.Encoding != "" && tag.Encoding != "utf8" {
				fc.encoder = encodeStringWithEncoding(tag.Encoding)
				fc.decoder = decodeStringWithEncoding(tag.Size, tag.Encoding)
			} else {
				fc.encoder = encodeString
				fc.decoder = decodeString(tag.Size)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported type: %v", field.Type)
	}

	return fc, nil
}
