package binpack

import "fmt"

// DecodeError 解码错误,包含详细的字段信息
type DecodeError struct {
	FieldName    string      // 字段名
	FieldType    string      // 字段类型
	Offset       int         // 字节偏移
	BitOffset    int         // 位偏移(-1表示非位字段)
	ExpectedSize int         // 期望长度(字节或位)
	ActualSize   int         // 实际长度(字节或位)
	Message      string      // 错误描述
	Cause        error       // 原始错误
}

func (e *DecodeError) Error() string {
	if e.BitOffset >= 0 {
		// 位字段错误
		return fmt.Sprintf(
			"field %q (%s) at offset %d bit %d: expected %d bits, got %d bits: %s",
			e.FieldName, e.FieldType, e.Offset, e.BitOffset,
			e.ExpectedSize, e.ActualSize, e.Message,
		)
	}
	// 普通字段错误
	return fmt.Sprintf(
		"field %q (%s) at offset %d: expected %d bytes, got %d bytes: %s",
		e.FieldName, e.FieldType, e.Offset,
		e.ExpectedSize, e.ActualSize, e.Message,
	)
}

func (e *DecodeError) Unwrap() error {
	return e.Cause
}

// EncodeError 编码错误
type EncodeError struct {
	FieldName string
	FieldType string
	Message   string
	Cause     error
}

func (e *EncodeError) Error() string {
	return fmt.Sprintf("field %q (%s): %s", e.FieldName, e.FieldType, e.Message)
}

func (e *EncodeError) Unwrap() error {
	return e.Cause
}

// newDecodeError 创建解码错误
func newDecodeError(fieldName, fieldType string, offset, expectedSize, actualSize int, message string) *DecodeError {
	return &DecodeError{
		FieldName:    fieldName,
		FieldType:    fieldType,
		Offset:       offset,
		BitOffset:    -1,
		ExpectedSize: expectedSize,
		ActualSize:   actualSize,
		Message:      message,
	}
}

// newBitDecodeError 创建位字段解码错误
func newBitDecodeError(fieldName, fieldType string, offset, bitOffset, expectedBits, actualBits int, message string) *DecodeError {
	return &DecodeError{
		FieldName:    fieldName,
		FieldType:    fieldType,
		Offset:       offset,
		BitOffset:    bitOffset,
		ExpectedSize: expectedBits,
		ActualSize:   actualBits,
		Message:      message,
	}
}

// newEncodeError 创建编码错误
func newEncodeError(fieldName, fieldType, message string) *EncodeError {
	return &EncodeError{
		FieldName: fieldName,
		FieldType: fieldType,
		Message:   message,
	}
}
