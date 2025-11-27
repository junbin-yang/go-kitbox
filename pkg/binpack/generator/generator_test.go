package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type TestPacket struct {
	Magic  uint32 `bin:"0:4:be"`
	Type   uint8  `bin:"4:1"`
	Length uint16 `bin:"5:2:le"`
	Data   uint32 `bin:"7:4:be"`
}

func TestGenerate(t *testing.T) {
	typ := reflect.TypeOf(TestPacket{})
	code, err := Generate(typ, "testpkg")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// 验证生成的代码包含必要的元素
	checks := []string{
		"package testpkg",
		"import \"encoding/binary\"",
		"func MarshalTestPacket",
		"func UnmarshalTestPacket",
		"binary.BigEndian.PutUint32",
		"binary.LittleEndian.PutUint16",
	}

	for _, check := range checks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("Generated code missing: %s", check)
		}
	}
}

func TestGenerateWithPointer(t *testing.T) {
	typ := reflect.TypeOf(&TestPacket{})
	code, err := Generate(typ, "testpkg")
	if err != nil {
		t.Fatalf("Generate with pointer failed: %v", err)
	}

	if len(code) == 0 {
		t.Error("Generated code is empty")
	}
}

func TestGenerateInvalidType(t *testing.T) {
	typ := reflect.TypeOf(123)
	_, err := Generate(typ, "testpkg")
	if err == nil {
		t.Error("Expected error for non-struct type")
	}
}

// TestGeneratedCodeCompiles 测试生成的代码是否可以编译和使用
func TestGeneratedCodeCompiles(t *testing.T) {
	// 生成代码
	typ := reflect.TypeOf(TestPacket{})
	code, err := Generate(typ, "testgen")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建 go.mod
	goMod := `module testgen

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// 写入生成的代码
	genFile := filepath.Join(tmpDir, "packet_gen.go")
	if err := os.WriteFile(genFile, code, 0644); err != nil {
		t.Fatalf("Failed to write generated code: %v", err)
	}

	// 写入测试代码
	testCode := `package testgen

import "testing"

func TestGeneratedMarshal(t *testing.T) {
	pkt := &TestPacket{
		Magic:  0x12345678,
		Type:   1,
		Length: 100,
		Data:   0xABCDEF00,
	}

	data, err := MarshalTestPacket(pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(data) != 11 {
		t.Errorf("Expected 11 bytes, got %d", len(data))
	}

	// 验证 Magic (大端序)
	if data[0] != 0x12 || data[1] != 0x34 || data[2] != 0x56 || data[3] != 0x78 {
		t.Errorf("Magic field incorrect")
	}

	// 验证 Type
	if data[4] != 1 {
		t.Errorf("Type field incorrect")
	}

	// 验证 Length (小端序)
	if data[5] != 100 || data[6] != 0 {
		t.Errorf("Length field incorrect")
	}
}

func TestGeneratedUnmarshal(t *testing.T) {
	data := []byte{
		0x12, 0x34, 0x56, 0x78, // Magic (BE)
		0x01,                   // Type
		0x64, 0x00,             // Length (LE)
		0xAB, 0xCD, 0xEF, 0x00, // Data (BE)
	}

	var pkt TestPacket
	if err := UnmarshalTestPacket(data, &pkt); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if pkt.Magic != 0x12345678 {
		t.Errorf("Magic: expected 0x12345678, got 0x%08X", pkt.Magic)
	}
	if pkt.Type != 1 {
		t.Errorf("Type: expected 1, got %d", pkt.Type)
	}
	if pkt.Length != 100 {
		t.Errorf("Length: expected 100, got %d", pkt.Length)
	}
	if pkt.Data != 0xABCDEF00 {
		t.Errorf("Data: expected 0xABCDEF00, got 0x%08X", pkt.Data)
	}
}

type TestPacket struct {
	Magic  uint32
	Type   uint8
	Length uint16
	Data   uint32
}
`
	testFile := filepath.Join(tmpDir, "packet_test.go")
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test code: %v", err)
	}

	// 运行测试
	cmd := exec.Command("go", "test", "-v")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Generated code test failed:\n%s\nError: %v", output, err)
	}

	t.Logf("Generated code test output:\n%s", output)
}
