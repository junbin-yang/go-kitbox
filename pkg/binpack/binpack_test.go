package binpack

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// TestBasicTypes 测试基础类型编解码
func TestBasicTypes(t *testing.T) {
	type Packet struct {
		U8  uint8   `bin:"0:1"`
		U16 uint16  `bin:"1:2:be"`
		U32 uint32  `bin:"3:4:be"`
		U64 uint64  `bin:"7:8:be"`
		I8  int8    `bin:"15:1"`
		I16 int16   `bin:"16:2:le"`
		I32 int32   `bin:"18:4:le"`
		I64 int64   `bin:"22:8:le"`
		F32 float32 `bin:"30:4:be"`
		F64 float64 `bin:"34:8:be"`
		B   bool    `bin:"42:1"`
	}

	pkt := Packet{
		U8:  0x12,
		U16: 0x1234,
		U32: 0x12345678,
		U64: 0x123456789ABCDEF0,
		I8:  -1,
		I16: -1000,
		I32: -100000,
		I64: -10000000000,
		F32: 3.14,
		F64: 2.718281828,
		B:   true,
	}

	// 编码
	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	if data[0] != 0x12 {
		t.Errorf("U8: expected 0x12, got 0x%02x", data[0])
	}
	if binary.BigEndian.Uint16(data[1:3]) != 0x1234 {
		t.Errorf("U16: expected 0x1234, got 0x%04x", binary.BigEndian.Uint16(data[1:3]))
	}
	if binary.BigEndian.Uint32(data[3:7]) != 0x12345678 {
		t.Errorf("U32: expected 0x12345678, got 0x%08x", binary.BigEndian.Uint32(data[3:7]))
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 验证解码结果
	if decoded.U8 != pkt.U8 {
		t.Errorf("U8: expected %d, got %d", pkt.U8, decoded.U8)
	}
	if decoded.U16 != pkt.U16 {
		t.Errorf("U16: expected %d, got %d", pkt.U16, decoded.U16)
	}
	if decoded.U32 != pkt.U32 {
		t.Errorf("U32: expected %d, got %d", pkt.U32, decoded.U32)
	}
	if decoded.U64 != pkt.U64 {
		t.Errorf("U64: expected %d, got %d", pkt.U64, decoded.U64)
	}
	if decoded.I8 != pkt.I8 {
		t.Errorf("I8: expected %d, got %d", pkt.I8, decoded.I8)
	}
	if decoded.I16 != pkt.I16 {
		t.Errorf("I16: expected %d, got %d", pkt.I16, decoded.I16)
	}
	if decoded.I32 != pkt.I32 {
		t.Errorf("I32: expected %d, got %d", pkt.I32, decoded.I32)
	}
	if decoded.I64 != pkt.I64 {
		t.Errorf("I64: expected %d, got %d", pkt.I64, decoded.I64)
	}
	if decoded.F32 != pkt.F32 {
		t.Errorf("F32: expected %f, got %f", pkt.F32, decoded.F32)
	}
	if decoded.F64 != pkt.F64 {
		t.Errorf("F64: expected %f, got %f", pkt.F64, decoded.F64)
	}
	if decoded.B != pkt.B {
		t.Errorf("B: expected %v, got %v", pkt.B, decoded.B)
	}
}

// TestByteOrder 测试字节序
func TestByteOrder(t *testing.T) {
	type Packet struct {
		BE uint32 `bin:"0:4:be"`
		LE uint32 `bin:"4:4:le"`
	}

	pkt := Packet{
		BE: 0x12345678,
		LE: 0x12345678,
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证大端序
	if binary.BigEndian.Uint32(data[0:4]) != 0x12345678 {
		t.Errorf("BE: expected 0x12345678, got 0x%08x", binary.BigEndian.Uint32(data[0:4]))
	}

	// 验证小端序
	if binary.LittleEndian.Uint32(data[4:8]) != 0x12345678 {
		t.Errorf("LE: expected 0x12345678, got 0x%08x", binary.LittleEndian.Uint32(data[4:8]))
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.BE != pkt.BE {
		t.Errorf("BE: expected %d, got %d", pkt.BE, decoded.BE)
	}
	if decoded.LE != pkt.LE {
		t.Errorf("LE: expected %d, got %d", pkt.LE, decoded.LE)
	}
}

// TestByteArray 测试固定长度字节数组
func TestByteArray(t *testing.T) {
	type Packet struct {
		Magic [4]byte `bin:"0:4"`
		Data  [8]byte `bin:"4:8"`
	}

	pkt := Packet{
		Magic: [4]byte{0x12, 0x34, 0x56, 0x78},
		Data:  [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	for i := 0; i < 4; i++ {
		if data[i] != pkt.Magic[i] {
			t.Errorf("Magic[%d]: expected 0x%02x, got 0x%02x", i, pkt.Magic[i], data[i])
		}
	}
	for i := 0; i < 8; i++ {
		if data[4+i] != pkt.Data[i] {
			t.Errorf("Data[%d]: expected 0x%02x, got 0x%02x", i, pkt.Data[i], data[4+i])
		}
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Magic != pkt.Magic {
		t.Errorf("Magic: expected %v, got %v", pkt.Magic, decoded.Magic)
	}
	if decoded.Data != pkt.Data {
		t.Errorf("Data: expected %v, got %v", pkt.Data, decoded.Data)
	}
}

// TestSkipField 测试跳过字段
func TestSkipField(t *testing.T) {
	type Packet struct {
		Magic    uint32 `bin:"0:4:be"`
		Internal string `bin:"-"`
		Version  uint8  `bin:"4:1"`
	}

	pkt := Packet{
		Magic:    0x12345678,
		Internal: "should be skipped",
		Version:  1,
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	if binary.BigEndian.Uint32(data[0:4]) != 0x12345678 {
		t.Errorf("Magic: expected 0x12345678, got 0x%08x", binary.BigEndian.Uint32(data[0:4]))
	}
	if data[4] != 1 {
		t.Errorf("Version: expected 1, got %d", data[4])
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Magic != pkt.Magic {
		t.Errorf("Magic: expected %d, got %d", pkt.Magic, decoded.Magic)
	}
	if decoded.Version != pkt.Version {
		t.Errorf("Version: expected %d, got %d", pkt.Version, decoded.Version)
	}
	if decoded.Internal != "" {
		t.Errorf("Internal: expected empty, got %s", decoded.Internal)
	}
}

// TestMarshalTo 测试编码到指定 buffer
func TestMarshalTo(t *testing.T) {
	type Packet struct {
		Magic   uint32 `bin:"0:4:be"`
		Version uint8  `bin:"4:1"`
	}

	pkt := Packet{
		Magic:   0x12345678,
		Version: 1,
	}

	buf := make([]byte, 100)
	n, err := MarshalTo(buf, &pkt)
	if err != nil {
		t.Fatalf("MarshalTo failed: %v", err)
	}

	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	// 验证编码结果
	if binary.BigEndian.Uint32(buf[0:4]) != 0x12345678 {
		t.Errorf("Magic: expected 0x12345678, got 0x%08x", binary.BigEndian.Uint32(buf[0:4]))
	}
	if buf[4] != 1 {
		t.Errorf("Version: expected 1, got %d", buf[4])
	}
}

// TestCodecCache 测试 codec 缓存
func TestCodecCache(t *testing.T) {
	type Packet struct {
		Magic uint32 `bin:"0:4:be"`
	}

	pkt := Packet{Magic: 0x12345678}

	// 第一次编码
	data1, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 第二次编码（应该使用缓存）
	data2, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证结果一致
	if len(data1) != len(data2) {
		t.Errorf("data length mismatch: %d vs %d", len(data1), len(data2))
	}
	for i := range data1 {
		if data1[i] != data2[i] {
			t.Errorf("data[%d] mismatch: 0x%02x vs 0x%02x", i, data1[i], data2[i])
		}
	}
}

// BenchmarkMarshal 性能测试：编码
func BenchmarkMarshal(b *testing.B) {
	type Packet struct {
		Magic   uint32 `bin:"0:4:be"`
		Type    uint8  `bin:"4:1"`
		Length  uint16 `bin:"5:2:le"`
		Payload [64]byte `bin:"7:64"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 64,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(&pkt)
	}
}

// BenchmarkUnmarshal 性能测试：解码
func BenchmarkUnmarshal(b *testing.B) {
	type Packet struct {
		Magic   uint32 `bin:"0:4:be"`
		Type    uint8  `bin:"4:1"`
		Length  uint16 `bin:"5:2:le"`
		Payload [64]byte `bin:"7:64"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 64,
	}

	data, _ := Marshal(&pkt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded Packet
		_ = Unmarshal(data, &decoded)
	}
}

// BenchmarkMarshalWithCodec 性能测试：使用预编译 codec 编码
func BenchmarkMarshalWithCodec(b *testing.B) {
	type Packet struct {
		Magic   uint32   `bin:"0:4:be"`
		Type    uint8    `bin:"4:1"`
		Length  uint16   `bin:"5:2:le"`
		Payload [64]byte `bin:"7:64"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 64,
	}

	codec := MustCompile(reflect.TypeOf(pkt))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = codec.Encode(&pkt)
	}
}

// TestMarshalWithPool 测试使用 buffer 池编码（零拷贝）
func TestMarshalWithPool(t *testing.T) {
	type Packet struct {
		Magic  uint32 `bin:"0:4:be"`
		Type   uint8  `bin:"4:1"`
		Length uint16 `bin:"5:2:le"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 7,
	}

	pool := NewBufferPool(128)
	data, err := MarshalWithPool(pool, &pkt)
	if err != nil {
		t.Fatalf("MarshalWithPool failed: %v", err)
	}
	defer pool.Put(&data)

	if len(data) != 7 {
		t.Errorf("Expected 7 bytes, got %d", len(data))
	}

	var decoded Packet
	if err := Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Magic != pkt.Magic || decoded.Type != pkt.Type || decoded.Length != pkt.Length {
		t.Error("Decoded data mismatch")
	}
}

// TestMarshalWithPoolCopy 测试使用 buffer 池编码（带复制）
func TestMarshalWithPoolCopy(t *testing.T) {
	type Packet struct {
		Magic  uint32 `bin:"0:4:be"`
		Type   uint8  `bin:"4:1"`
		Length uint16 `bin:"5:2:le"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 7,
	}

	pool := NewBufferPool(128)
	data, err := MarshalWithPoolCopy(pool, &pkt)
	if err != nil {
		t.Fatalf("MarshalWithPoolCopy failed: %v", err)
	}

	if len(data) != 7 {
		t.Errorf("Expected 7 bytes, got %d", len(data))
	}

	var decoded Packet
	if err := Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Magic != pkt.Magic || decoded.Type != pkt.Type || decoded.Length != pkt.Length {
		t.Error("Decoded data mismatch")
	}
}

// BenchmarkMarshalWithPool 性能测试：使用 buffer 池编码（零拷贝）
func BenchmarkMarshalWithPool(b *testing.B) {
	type Packet struct {
		Magic   uint32   `bin:"0:4:be"`
		Type    uint8    `bin:"4:1"`
		Length  uint16   `bin:"5:2:le"`
		Payload [64]byte `bin:"7:64"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 64,
	}

	pool := NewBufferPool(128)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := MarshalWithPool(pool, &pkt)
		pool.Put(&data)
	}
}

// BenchmarkMarshalWithPoolCopy 性能测试：使用 buffer 池编码（带复制）
func BenchmarkMarshalWithPoolCopy(b *testing.B) {
	type Packet struct {
		Magic   uint32   `bin:"0:4:be"`
		Type    uint8    `bin:"4:1"`
		Length  uint16   `bin:"5:2:le"`
		Payload [64]byte `bin:"7:64"`
	}

	pkt := Packet{
		Magic:  0x12345678,
		Type:   1,
		Length: 64,
	}

	pool := NewBufferPool(128)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalWithPoolCopy(pool, &pkt)
	}
}

// TestVariableLengthFields 测试变长字段
func TestVariableLengthFields(t *testing.T) {
	type Packet struct {
		Magic   uint32 `bin:"0:4:be"`
		Length  uint16 `bin:"4:2:le"`
		Payload []byte `bin:"6:var,len:Length"`
	}

	payload := []byte("Hello, World!")
	pkt := Packet{
		Magic:   0x12345678,
		Length:  uint16(len(payload)),
		Payload: payload,
	}

	// 编码
	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	if binary.BigEndian.Uint32(data[0:4]) != 0x12345678 {
		t.Errorf("Magic: expected 0x12345678, got 0x%08x", binary.BigEndian.Uint32(data[0:4]))
	}
	if binary.LittleEndian.Uint16(data[4:6]) != uint16(len(payload)) {
		t.Errorf("Length: expected %d, got %d", len(payload), binary.LittleEndian.Uint16(data[4:6]))
	}
	if string(data[6:6+len(payload)]) != string(payload) {
		t.Errorf("Payload: expected %s, got %s", payload, data[6:6+len(payload)])
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Magic != pkt.Magic {
		t.Errorf("Magic: expected %d, got %d", pkt.Magic, decoded.Magic)
	}
	if decoded.Length != pkt.Length {
		t.Errorf("Length: expected %d, got %d", pkt.Length, decoded.Length)
	}
	if string(decoded.Payload) != string(pkt.Payload) {
		t.Errorf("Payload: expected %s, got %s", pkt.Payload, decoded.Payload)
	}
}

// TestVariableLengthString 测试变长字符串
func TestVariableLengthString(t *testing.T) {
	type Packet struct {
		Magic  uint32 `bin:"0:4:be"`
		Length uint16 `bin:"4:2:le"`
		Name   string `bin:"6:var,len:Length"`
	}

	name := "Alice"
	pkt := Packet{
		Magic:  0x12345678,
		Length: uint16(len(name)),
		Name:   name,
	}

	// 编码
	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	if string(data[6:6+len(name)]) != name {
		t.Errorf("Name: expected %s, got %s", name, data[6:6+len(name)])
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Name != pkt.Name {
		t.Errorf("Name: expected %s, got %s", pkt.Name, decoded.Name)
	}
}

// TestFixedLengthString 测试固定长度字符串
func TestFixedLengthString(t *testing.T) {
	type Packet struct {
		Magic uint32 `bin:"0:4:be"`
		Name  string `bin:"4:16"`
	}

	pkt := Packet{
		Magic: 0x12345678,
		Name:  "Alice",
	}

	// 编码
	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	if string(data[4:9]) != "Alice" {
		t.Errorf("Name: expected Alice, got %s", data[4:9])
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 固定长度字符串会包含填充的零字节
	if decoded.Name[:5] != pkt.Name {
		t.Errorf("Name: expected %s, got %s", pkt.Name, decoded.Name[:5])
	}
}

// TestBitFields 测试位字段
func TestBitFields(t *testing.T) {
	type Packet struct {
		Flags uint8 `bin:"0:1"`
		Bit0  uint8 `bin:"1:1,bits:0"`   // 位 0
		Bit12 uint8 `bin:"1:1,bits:1-2"` // 位 1-2
		Bit34 uint8 `bin:"1:1,bits:3-4"` // 位 3-4
		Bit5  uint8 `bin:"1:1,bits:5"`   // 位 5
	}

	pkt := Packet{
		Flags: 0xFF,
		Bit0:  1,   // 0b1
		Bit12: 2,   // 0b10
		Bit34: 3,   // 0b11
		Bit5:  1,   // 0b1
	}

	// 编码
	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证编码结果
	// Bit0=1, Bit12=2(0b10), Bit34=3(0b11), Bit5=1
	// 字节1 = 0b00111101 = 0x3D (bit5=1, bit4-3=11, bit2-1=10, bit0=1)
	expected := byte(0b00111101)
	if data[1] != expected {
		t.Errorf("Bit fields: expected 0x%02x, got 0x%02x", expected, data[1])
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Bit0 != pkt.Bit0 {
		t.Errorf("Bit0: expected %d, got %d", pkt.Bit0, decoded.Bit0)
	}
	if decoded.Bit12 != pkt.Bit12 {
		t.Errorf("Bit12: expected %d, got %d", pkt.Bit12, decoded.Bit12)
	}
	if decoded.Bit34 != pkt.Bit34 {
		t.Errorf("Bit34: expected %d, got %d", pkt.Bit34, decoded.Bit34)
	}
	if decoded.Bit5 != pkt.Bit5 {
		t.Errorf("Bit5: expected %d, got %d", pkt.Bit5, decoded.Bit5)
	}
}

// TestConditionalFields 测试条件字段
func TestConditionalFields(t *testing.T) {
	type Packet struct {
		Type    uint8  `bin:"0:1"`
		Length  uint16 `bin:"1:2:le"`
		Data1   uint32 `bin:"3:4:be,if:Type==1"` // 仅当 Type==1 时编码
		Data2   uint32 `bin:"3:4:be,if:Type==2"` // 仅当 Type==2 时编码
	}

	// 测试 Type==1
	pkt1 := Packet{
		Type:  1,
		Length: 100,
		Data1: 0x12345678,
		Data2: 0xABCDEF00, // 不会被编码
	}

	data1, err := Marshal(&pkt1)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证 Data1 被编码
	if binary.BigEndian.Uint32(data1[3:7]) != 0x12345678 {
		t.Errorf("Data1: expected 0x12345678, got 0x%08x", binary.BigEndian.Uint32(data1[3:7]))
	}

	// 解码
	var decoded1 Packet
	err = Unmarshal(data1, &decoded1)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded1.Data1 != pkt1.Data1 {
		t.Errorf("Data1: expected %d, got %d", pkt1.Data1, decoded1.Data1)
	}
	if decoded1.Data2 != 0 {
		t.Errorf("Data2: expected 0, got %d", decoded1.Data2)
	}

	// 测试 Type==2
	pkt2 := Packet{
		Type:  2,
		Length: 200,
		Data1: 0x12345678, // 不会被编码
		Data2: 0xABCDEF00,
	}

	data2, err := Marshal(&pkt2)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证 Data2 被编码
	if binary.BigEndian.Uint32(data2[3:7]) != 0xABCDEF00 {
		t.Errorf("Data2: expected 0xABCDEF00, got 0x%08x", binary.BigEndian.Uint32(data2[3:7]))
	}

	// 解码
	var decoded2 Packet
	err = Unmarshal(data2, &decoded2)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded2.Data1 != 0 {
		t.Errorf("Data1: expected 0, got %d", decoded2.Data1)
	}
	if decoded2.Data2 != pkt2.Data2 {
		t.Errorf("Data2: expected %d, got %d", pkt2.Data2, decoded2.Data2)
	}
}

// TestStringEncoding 测试字符串编码
func TestStringEncoding(t *testing.T) {
	type Packet struct {
		Magic    uint32 `bin:"0:4:be"`
		UTF8Data string `bin:"4:10"`           // UTF-8 编码（默认）
		ASCIIData string `bin:"14:10,enc:ascii"` // ASCII 编码
		HexData  string `bin:"24:10,enc:hex"`  // Hex 编码
	}

	pkt := Packet{
		Magic:     0x12345678,
		UTF8Data:  "Hello",     // 5 字节
		ASCIIData: "World",     // 5 字节
		HexData:   "Test!",     // 5 字节
	}

	// 编码
	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证 UTF-8 编码结果
	expectedUTF8 := "Hello"
	actualUTF8 := string(data[4:9])
	if actualUTF8 != expectedUTF8 {
		t.Errorf("UTF8Data: expected %s, got %s", expectedUTF8, actualUTF8)
	}

	// 验证 ASCII 编码结果
	expectedASCII := "World"
	actualASCII := string(data[14:19])
	if actualASCII != expectedASCII {
		t.Errorf("ASCIIData: expected %s, got %s", expectedASCII, actualASCII)
	}

	// 验证 Hex 编码结果 - "Test!" 的十六进制是 "5465737421"
	expectedHex := "5465737421"
	actualHex := string(data[24:34])
	if actualHex != expectedHex {
		t.Errorf("HexData: expected %s, got %s", expectedHex, actualHex)
	}

	// 解码
	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 验证解码结果
	if decoded.UTF8Data[:5] != pkt.UTF8Data {
		t.Errorf("UTF8Data: expected %s, got %s", pkt.UTF8Data, decoded.UTF8Data[:5])
	}
	if decoded.ASCIIData[:5] != pkt.ASCIIData {
		t.Errorf("ASCIIData: expected %s, got %s", pkt.ASCIIData, decoded.ASCIIData[:5])
	}
	if decoded.HexData != pkt.HexData {
		t.Errorf("HexData: expected %s, got %s", pkt.HexData, decoded.HexData)
	}
}
