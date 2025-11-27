package binpack

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sync"
)

// reflectCodec 基于反射的编解码器实现
type reflectCodec struct {
	typ        reflect.Type
	fields     []*fieldCodec
	size       int
	fieldMap   map[string]int // 字段名到索引的映射
	hasVarLen  bool           // 是否包含变长字段
}

// codecCache 缓存已编译的 codec
var codecCache sync.Map

// CompileCodec 编译类型的 codec
func CompileCodec(typ reflect.Type) (Codec, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type must be struct, got %v", typ.Kind())
	}

	// 检查缓存
	if cached, ok := codecCache.Load(typ); ok {
		return cached.(Codec), nil
	}

	codec := &reflectCodec{
		typ:      typ,
		fields:   make([]*fieldCodec, 0),
		fieldMap: make(map[string]int),
	}

	// 遍历结构体字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		codec.fieldMap[field.Name] = i

		// 获取 bin tag
		tag := field.Tag.Get("bin")
		if tag == "" {
			continue
		}

		// 解析 tag
		tagInfo, err := parseTag(tag)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		// 跳过字段
		if tagInfo.Skip {
			continue
		}

		// 构建字段编解码器
		fc, err := buildFieldCodec(field, tagInfo)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		if fc != nil {
			fc.index = i
			codec.fields = append(codec.fields, fc)

			if fc.isVariable || fc.isRepeat {
				codec.hasVarLen = true
			}

			// 计算总大小（仅固定长度字段）
			if tagInfo.Offset >= 0 && tagInfo.Size > 0 {
				end := tagInfo.Offset + tagInfo.Size
				if end > codec.size {
					codec.size = end
				}
			}
		}
	}

	// 解析变长字段的长度字段索引和条件字段索引
	for _, fc := range codec.fields {
		if fc.isVariable && fc.lenField != "" {
			lenIdx, ok := codec.fieldMap[fc.lenField]
			if !ok {
				return nil, fmt.Errorf("length field %s not found", fc.lenField)
			}
			fc.lenIndex = lenIdx
		}
		if fc.conditional && fc.condField != "" {
			condIdx, ok := codec.fieldMap[fc.condField]
			if !ok {
				return nil, fmt.Errorf("condition field %s not found", fc.condField)
			}
			fc.condIndex = condIdx
		}
	}

	// 缓存 codec
	codecCache.Store(typ, codec)

	return codec, nil
}

// MustCompile 编译类型的 codec，失败时 panic
func MustCompile(typ reflect.Type) Codec {
	codec, err := CompileCodec(typ)
	if err != nil {
		panic(err)
	}
	return codec
}

// Encode 编码结构体为字节流
func (c *reflectCodec) Encode(v interface{}) ([]byte, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Type() != c.typ {
		return nil, fmt.Errorf("type mismatch: expected %v, got %v", c.typ, val.Type())
	}

	// 计算总大小
	totalSize := c.size
	if c.hasVarLen {
		for _, fc := range c.fields {
			fieldVal := val.Field(fc.index)
			if fc.isRepeat && fieldVal.Kind() == reflect.Slice {
				// 数组字段
				count := fieldVal.Len()
				if fc.elementSize > 0 {
					// 基础类型数组
					end := fc.offset + count*fc.elementSize
					if end > totalSize {
						totalSize = end
					}
				} else {
					// 结构体数组，需要计算每个元素的大小
					elemSize := 0
					for i := 0; i < count; i++ {
						elem := fieldVal.Index(i)
						elemCodec, _ := CompileCodec(elem.Type())
						if elemCodec != nil {
							elemData, _ := elemCodec.Encode(elem.Addr().Interface())
							elemSize += len(elemData)
						}
					}
					end := fc.offset + elemSize
					if end > totalSize {
						totalSize = end
					}
				}
			} else if fc.isVariable {
				varLen := 0
				if fieldVal.Kind() == reflect.Slice {
					varLen = fieldVal.Len()
				} else if fieldVal.Kind() == reflect.String {
					varLen = len(fieldVal.String())
				}
				end := fc.offset + varLen
				if end > totalSize {
					totalSize = end
				}
			}
		}
	}

	buf := make([]byte, totalSize)
	_, err := c.encodeToBuffer(buf, val)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// EncodeTo 编码结构体到指定 buffer
func (c *reflectCodec) EncodeTo(buf []byte, v interface{}) (int, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Type() != c.typ {
		return 0, fmt.Errorf("type mismatch: expected %v, got %v", c.typ, val.Type())
	}

	return c.encodeToBuffer(buf, val)
}

// encodeToBuffer 内部编码函数
func (c *reflectCodec) encodeToBuffer(buf []byte, val reflect.Value) (int, error) {
	maxSize := c.size

	// 编码每个字段
	for _, fc := range c.fields {
		// 检查条件字段
		if fc.conditional {
			condFieldVal := val.Field(fc.condIndex)
			if condFieldVal.Uint() != fc.condValue {
				continue // 跳过不满足条件的字段
			}
		}

		fieldVal := val.Field(fc.index)

		if fc.isRepeat && fieldVal.Kind() == reflect.Slice {
			// 数组字段
			count := fieldVal.Len()
			currentOffset := fc.offset

			if fc.elementSize > 0 {
				// 基础类型数组
				totalSize := count * fc.elementSize
				if fc.offset+totalSize > len(buf) {
					return 0, fmt.Errorf("buffer too small for array field %d", fc.index)
				}

				for i := 0; i < count; i++ {
					elem := fieldVal.Index(i)
					offset := fc.offset + i*fc.elementSize
					if err := encodeElement(buf[offset:offset+fc.elementSize], elem, fc.elementSize, fc.byteOrder); err != nil {
						return 0, fmt.Errorf("encode array element %d: %w", i, err)
					}
				}
				currentOffset += totalSize
			} else {
				// 结构体数组
				for i := 0; i < count; i++ {
					elem := fieldVal.Index(i)
					codec, err := CompileCodec(elem.Type())
					if err != nil {
						return 0, fmt.Errorf("compile codec for array element %d: %w", i, err)
					}
					n, err := codec.EncodeTo(buf[currentOffset:], elem.Addr().Interface())
					if err != nil {
						return 0, fmt.Errorf("encode array element %d: %w", i, err)
					}
					currentOffset += n
				}
			}

			if currentOffset > maxSize {
				maxSize = currentOffset
			}
		} else if fc.isVariable {
			// 变长字段
			varLen := 0
			if fieldVal.Kind() == reflect.Slice {
				varLen = fieldVal.Len()
			} else if fieldVal.Kind() == reflect.String {
				varLen = len(fieldVal.String())
			}

			if fc.offset+varLen > len(buf) {
				return 0, fmt.Errorf("buffer too small for variable field %d", fc.index)
			}

			if err := fc.encoder(buf[fc.offset:fc.offset+varLen], fieldVal); err != nil {
				return 0, fmt.Errorf("encode field %d: %w", fc.index, err)
			}

			end := fc.offset + varLen
			if end > maxSize {
				maxSize = end
			}
		} else {
			// 固定长度字段
			if err := fc.encoder(buf[fc.offset:fc.offset+fc.size], fieldVal); err != nil {
				return 0, fmt.Errorf("encode field %d: %w", fc.index, err)
			}
		}
	}

	return maxSize, nil
}

// Decode 解码字节流为结构体
func (c *reflectCodec) Decode(data []byte, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}

	val = val.Elem()
	if val.Type() != c.typ {
		return fmt.Errorf("type mismatch: expected %v, got %v", c.typ, val.Type())
	}

	if len(data) < c.size {
		return newDecodeError("", c.typ.String(), 0, c.size, len(data), "data too short")
	}

	// 解码每个字段
	for _, fc := range c.fields {
		// 检查条件字段
		if fc.conditional {
			condFieldVal := val.Field(fc.condIndex)
			if condFieldVal.Uint() != fc.condValue {
				continue // 跳过不满足条件的字段
			}
		}

		fieldVal := val.Field(fc.index)

		if fc.isRepeat && fieldVal.Kind() == reflect.Slice {
			// 数组字段
			lenFieldVal := val.Field(fc.lenIndex)
			count := int(lenFieldVal.Uint())

			// 创建切片
			slice := reflect.MakeSlice(fieldVal.Type(), count, count)

			if fc.elementSize > 0 {
				// 基础类型数组
				totalSize := count * fc.elementSize
				if fc.offset+totalSize > len(data) {
					return newDecodeError(fc.name, fc.typeName, fc.offset, totalSize, len(data)-fc.offset, "data too short for array field")
				}

				for i := 0; i < count; i++ {
					elem := slice.Index(i)
					offset := fc.offset + i*fc.elementSize
					if err := decodeElement(data[offset:offset+fc.elementSize], elem, fc.elementSize, fc.byteOrder); err != nil {
						return fmt.Errorf("decode array element %d: %w", i, err)
					}
				}
			} else {
				// 结构体数组
				currentOffset := fc.offset
				for i := 0; i < count; i++ {
					elem := slice.Index(i)
					codec, err := CompileCodec(elem.Type())
					if err != nil {
						return fmt.Errorf("compile codec for array element %d: %w", i, err)
					}
					if err := codec.Decode(data[currentOffset:], elem.Addr().Interface()); err != nil {
						return fmt.Errorf("decode array element %d: %w", i, err)
					}
					// 计算元素大小以更新偏移
					elemData, _ := codec.Encode(elem.Addr().Interface())
					currentOffset += len(elemData)
				}
			}

			fieldVal.Set(slice)
		} else if fc.isVariable {
			// 变长字段，需要从长度字段获取长度
			lenFieldVal := val.Field(fc.lenIndex)
			varLen := int(lenFieldVal.Uint())

			if fc.offset+varLen > len(data) {
				return newDecodeError(fc.name, fc.typeName, fc.offset, varLen, len(data)-fc.offset, "data too short for variable field")
			}

			// 动态创建解码器
			if fieldVal.Kind() == reflect.Slice {
				slice := make([]byte, varLen)
				copy(slice, data[fc.offset:fc.offset+varLen])
				fieldVal.Set(reflect.ValueOf(slice))
			} else if fieldVal.Kind() == reflect.String {
				// 根据编码方式解码字符串
				if fc.encoding == "hex" {
					dst := make([]byte, varLen/2)
					for i := 0; i < len(dst) && i*2+1 < varLen; i++ {
						hi := hexCharToNibble(data[fc.offset+i*2])
						lo := hexCharToNibble(data[fc.offset+i*2+1])
						dst[i] = (hi << 4) | lo
					}
					fieldVal.SetString(string(dst))
				} else {
					fieldVal.SetString(string(data[fc.offset : fc.offset+varLen]))
				}
			}
		} else {
			// 固定长度字段
			if err := fc.decoder(data[fc.offset:fc.offset+fc.size], fieldVal); err != nil {
				return fmt.Errorf("decode field %d: %w", fc.index, err)
			}
		}
	}

	return nil
}

// encodeElement 编码单个数组元素
func encodeElement(buf []byte, v reflect.Value, size int, byteOrder binary.ByteOrder) error {
	switch v.Kind() {
	case reflect.Uint8:
		buf[0] = uint8(v.Uint())
	case reflect.Uint16:
		byteOrder.PutUint16(buf, uint16(v.Uint()))
	case reflect.Uint32:
		byteOrder.PutUint32(buf, uint32(v.Uint()))
	case reflect.Uint64:
		byteOrder.PutUint64(buf, v.Uint())
	case reflect.Int8:
		buf[0] = uint8(v.Int())
	case reflect.Int16:
		byteOrder.PutUint16(buf, uint16(v.Int()))
	case reflect.Int32:
		byteOrder.PutUint32(buf, uint32(v.Int()))
	case reflect.Int64:
		byteOrder.PutUint64(buf, uint64(v.Int()))
	case reflect.Float32:
		byteOrder.PutUint32(buf, math.Float32bits(float32(v.Float())))
	case reflect.Float64:
		byteOrder.PutUint64(buf, math.Float64bits(v.Float()))
	case reflect.Struct:
		// 结构体数组元素
		codec, err := CompileCodec(v.Type())
		if err != nil {
			return err
		}
		_, err = codec.EncodeTo(buf, v.Addr().Interface())
		return err
	default:
		return fmt.Errorf("unsupported element type: %v", v.Kind())
	}
	return nil
}

// decodeElement 解码单个数组元素
func decodeElement(data []byte, v reflect.Value, size int, byteOrder binary.ByteOrder) error {
	switch v.Kind() {
	case reflect.Uint8:
		v.SetUint(uint64(data[0]))
	case reflect.Uint16:
		v.SetUint(uint64(byteOrder.Uint16(data)))
	case reflect.Uint32:
		v.SetUint(uint64(byteOrder.Uint32(data)))
	case reflect.Uint64:
		v.SetUint(byteOrder.Uint64(data))
	case reflect.Int8:
		v.SetInt(int64(int8(data[0])))
	case reflect.Int16:
		v.SetInt(int64(int16(byteOrder.Uint16(data))))
	case reflect.Int32:
		v.SetInt(int64(int32(byteOrder.Uint32(data))))
	case reflect.Int64:
		v.SetInt(int64(byteOrder.Uint64(data)))
	case reflect.Float32:
		v.SetFloat(float64(math.Float32frombits(byteOrder.Uint32(data))))
	case reflect.Float64:
		v.SetFloat(math.Float64frombits(byteOrder.Uint64(data)))
	case reflect.Struct:
		// 结构体数组元素
		codec, err := CompileCodec(v.Type())
		if err != nil {
			return err
		}
		return codec.Decode(data, v.Addr().Interface())
	default:
		return fmt.Errorf("unsupported element type: %v", v.Kind())
	}
	return nil
}
