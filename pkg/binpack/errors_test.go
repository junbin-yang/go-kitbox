package binpack

import (
	"errors"
	"testing"
)

func TestDecodeError(t *testing.T) {
	// 测试普通字段错误
	err := newDecodeError("MessageID", "uint16", 16, 2, 1, "data too short")
	expected := `field "MessageID" (uint16) at offset 16: expected 2 bytes, got 1 bytes: data too short`
	if err.Error() != expected {
		t.Errorf("错误消息不匹配:\ngot:  %s\nwant: %s", err.Error(), expected)
	}

	// 测试位字段错误
	bitErr := newBitDecodeError("Enable", "uint8", 1, 0, 1, 0, "bit not set")
	expected = `field "Enable" (uint8) at offset 1 bit 0: expected 1 bits, got 0 bits: bit not set`
	if bitErr.Error() != expected {
		t.Errorf("位字段错误消息不匹配:\ngot:  %s\nwant: %s", bitErr.Error(), expected)
	}
}

func TestEncodeError(t *testing.T) {
	err := newEncodeError("Payload", "[]byte", "length exceeds maximum")
	expected := `field "Payload" ([]byte): length exceeds maximum`
	if err.Error() != expected {
		t.Errorf("编码错误消息不匹配:\ngot:  %s\nwant: %s", err.Error(), expected)
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("原始错误")
	err := &DecodeError{
		FieldName: "Test",
		FieldType: "uint32",
		Cause:     cause,
	}

	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Error("Unwrap 未返回原始错误")
	}
}
