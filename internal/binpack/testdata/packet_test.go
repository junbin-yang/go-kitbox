package testdata

import (
	"reflect"
	"testing"

	"github.com/junbin-yang/go-kitbox/pkg/binpack"
)

func TestGeneratedMarshalPacket(t *testing.T) {
	pkt := &Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 100,
	}

	data, err := MarshalPacket(pkt)
	if err != nil {
		t.Fatalf("MarshalPacket failed: %v", err)
	}

	if len(data) != 7 {
		t.Errorf("Expected 7 bytes, got %d", len(data))
	}

	// 验证 Magic (大端序)
	if data[0] != 0x12 || data[1] != 0x34 || data[2] != 0x56 || data[3] != 0x78 {
		t.Errorf("Magic field incorrect: got %x %x %x %x", data[0], data[1], data[2], data[3])
	}

	// 验证 Type
	if data[4] != 1 {
		t.Errorf("Type field incorrect: got %d", data[4])
	}

	// 验证 Length (小端序)
	if data[5] != 100 || data[6] != 0 {
		t.Errorf("Length field incorrect: got %d %d", data[5], data[6])
	}
}

func TestGeneratedUnmarshalPacket(t *testing.T) {
	data := []byte{
		0x12, 0x34, 0x56, 0x78, // Magic (BE)
		0x01,                   // Type
		0x64, 0x00,             // Length (LE) = 100
	}

	var pkt Packet
	if err := UnmarshalPacket(data, &pkt); err != nil {
		t.Fatalf("UnmarshalPacket failed: %v", err)
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
}

func TestGeneratedRoundTrip(t *testing.T) {
	original := &Packet{
		Magic:  0xABCDEF01,
		Type:   255,
		Length: 65535,
	}

	// Marshal
	data, err := MarshalPacket(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded Packet
	if err := UnmarshalPacket(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 验证
	if decoded.Magic != original.Magic {
		t.Errorf("Magic mismatch: expected 0x%08X, got 0x%08X", original.Magic, decoded.Magic)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: expected %d, got %d", original.Type, decoded.Type)
	}
	if decoded.Length != original.Length {
		t.Errorf("Length mismatch: expected %d, got %d", original.Length, decoded.Length)
	}
}

// 性能对比基准测试
var (
	testPacket = &Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 100,
	}
	packetCodec = binpack.MustCompile(reflect.TypeOf(Packet{}))
)

// BenchmarkReflectionMode 反射模式
func BenchmarkReflectionMode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = binpack.Marshal(testPacket)
	}
}

// BenchmarkPrecompiledMode 预编译模式
func BenchmarkPrecompiledMode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = packetCodec.Encode(testPacket)
	}
}

// BenchmarkCodeGenMode 代码生成模式
func BenchmarkCodeGenMode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalPacket(testPacket)
	}
}
