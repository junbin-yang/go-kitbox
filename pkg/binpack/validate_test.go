package binpack

import (
	"strings"
	"testing"
)

func TestValidateStruct_Valid(t *testing.T) {
	type ValidPacket struct {
		Length uint16 `bin:"0:2:be"`
		Type   uint8  `bin:"2:1"`
		Data   []byte `bin:"3:var,len:Length"`
	}

	if err := ValidateStruct(ValidPacket{}); err != nil {
		t.Errorf("expected valid struct, got error: %v", err)
	}
}

func TestValidateStruct_OffsetConflict(t *testing.T) {
	type ConflictPacket struct {
		Field1 uint16 `bin:"0:2:be"`
		Field2 uint8  `bin:"0:1"` // 冲突：偏移量 0 已被 Field1 使用
	}

	err := ValidateStruct(ConflictPacket{})
	if err == nil {
		t.Error("expected offset conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Errorf("expected conflict error, got: %v", err)
	}
}

func TestValidateStruct_MissingLenField(t *testing.T) {
	type MissingLenPacket struct {
		Type uint8  `bin:"0:1"`
		Data []byte `bin:"1:var,len:Length"` // Length 字段不存在
	}

	err := ValidateStruct(MissingLenPacket{})
	if err == nil {
		t.Error("expected missing length field error, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected missing field error, got: %v", err)
	}
}

func TestValidateStruct_LenFieldAfter(t *testing.T) {
	type WrongOrderPacket struct {
		Data   []byte `bin:"0:var,len:Length"` // Length 在后面
		Length uint16 `bin:"10:2:be"`
	}

	err := ValidateStruct(WrongOrderPacket{})
	if err == nil {
		t.Error("expected length field order error, got nil")
	}
	if !strings.Contains(err.Error(), "must appear before") {
		t.Errorf("expected order error, got: %v", err)
	}
}

func TestValidateStruct_MissingCondField(t *testing.T) {
	type MissingCondPacket struct {
		Type uint8  `bin:"0:1"`
		Data []byte `bin:"1:4,if:Status==1"` // Status 字段不存在
	}

	err := ValidateStruct(MissingCondPacket{})
	if err == nil {
		t.Error("expected missing condition field error, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected missing field error, got: %v", err)
	}
}

func TestValidateStruct_VarWithoutLen(t *testing.T) {
	type NoLenPacket struct {
		Type uint8  `bin:"0:1"`
		Data []byte `bin:"1:var"` // 变长字段没有指定 len
	}

	err := ValidateStruct(NoLenPacket{})
	if err == nil {
		t.Error("expected variable-length without len error, got nil")
	}
	if !strings.Contains(err.Error(), "must specify len") {
		t.Errorf("expected len requirement error, got: %v", err)
	}
}

func TestValidateStruct_RepeatWithoutLen(t *testing.T) {
	type NoLenRepeatPacket struct {
		Type  uint8    `bin:"0:1"`
		Items []uint16 `bin:"1:2:be,repeat"` // repeat 没有指定 len
	}

	err := ValidateStruct(NoLenRepeatPacket{})
	if err == nil {
		t.Error("expected repeat without len error, got nil")
	}
	if !strings.Contains(err.Error(), "must specify len") {
		t.Errorf("expected len requirement error, got: %v", err)
	}
}

func TestValidateStruct_Overlap(t *testing.T) {
	type OverlapPacket struct {
		Field1 uint32 `bin:"0:4:be"` // 占用 0-3
		Field2 uint16 `bin:"2:2:be"` // 占用 2-3，与 Field1 重叠
	}

	err := ValidateStruct(OverlapPacket{})
	if err == nil {
		t.Error("expected overlap error, got nil")
	}
	if !strings.Contains(err.Error(), "overlaps") {
		t.Errorf("expected overlap error, got: %v", err)
	}
}

func TestValidateStruct_ComplexValid(t *testing.T) {
	type ComplexPacket struct {
		Length uint16   `bin:"0:2:be"`
		Type   uint8    `bin:"2:1"`
		Count  uint8    `bin:"3:1"`
		Items  []uint16 `bin:"4:2:be,repeat,len:Count"`
		Data   []byte   `bin:"-1:var,len:Length"`
	}

	if err := ValidateStruct(ComplexPacket{}); err != nil {
		t.Errorf("expected valid complex struct, got error: %v", err)
	}
}
