package binpack

import (
	"reflect"
	"testing"
)

// TestArrayField_BasicTypes 测试基础类型数组
func TestArrayField_BasicTypes(t *testing.T) {
	type ModbusPacket struct {
		FunctionCode  uint8    `bin:"0:1"`
		RegisterCount uint8    `bin:"1:1"`
		Registers     []uint16 `bin:"2:var,len:RegisterCount,repeat,size:2:be"`
	}

	tests := []struct {
		name    string
		packet  ModbusPacket
		want    []byte
		wantErr bool
	}{
		{
			name: "3个寄存器",
			packet: ModbusPacket{
				FunctionCode:  0x03,
				RegisterCount: 3,
				Registers:     []uint16{100, 200, 300},
			},
			want: []byte{0x03, 0x03, 0x00, 0x64, 0x00, 0xC8, 0x01, 0x2C},
		},
		{
			name: "空数组",
			packet: ModbusPacket{
				FunctionCode:  0x03,
				RegisterCount: 0,
				Registers:     []uint16{},
			},
			want: []byte{0x03, 0x00},
		},
		{
			name: "单个寄存器",
			packet: ModbusPacket{
				FunctionCode:  0x03,
				RegisterCount: 1,
				Registers:     []uint16{0xABCD},
			},
			want: []byte{0x03, 0x01, 0xAB, 0xCD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试编码
			got, err := Marshal(&tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Marshal() = %v, want %v", got, tt.want)
			}

			// 测试解码
			if !tt.wantErr {
				var decoded ModbusPacket
				err = Unmarshal(got, &decoded)
				if err != nil {
					t.Errorf("Unmarshal() error = %v", err)
					return
				}
				if !reflect.DeepEqual(decoded, tt.packet) {
					t.Errorf("Unmarshal() = %+v, want %+v", decoded, tt.packet)
				}
			}
		})
	}
}

// TestArrayField_LittleEndian 测试小端序数组
func TestArrayField_LittleEndian(t *testing.T) {
	type Packet struct {
		Count  uint8    `bin:"0:1"`
		Values []uint32 `bin:"1:var,len:Count,repeat,size:4:le"`
	}

	pkt := Packet{
		Count:  2,
		Values: []uint32{0x12345678, 0xABCDEF00},
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := []byte{0x02, 0x78, 0x56, 0x34, 0x12, 0x00, 0xEF, 0xCD, 0xAB}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("Marshal() = %v, want %v", data, want)
	}

	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, pkt) {
		t.Errorf("Unmarshal() = %+v, want %+v", decoded, pkt)
	}
}

// TestArrayField_SignedIntegers 测试有符号整数数组
func TestArrayField_SignedIntegers(t *testing.T) {
	type Packet struct {
		Count  uint8   `bin:"0:1"`
		Values []int16 `bin:"1:var,len:Count,repeat,size:2:be"`
	}

	pkt := Packet{
		Count:  3,
		Values: []int16{-100, 0, 100},
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, pkt) {
		t.Errorf("Unmarshal() = %+v, want %+v", decoded, pkt)
	}
}

// TestArrayField_Struct 测试结构体数组
func TestArrayField_Struct(t *testing.T) {
	type Option struct {
		Delta  uint8  `bin:"0:1"`
		Length uint8  `bin:"1:1"`
		Value  []byte `bin:"2:var,len:Length"`
	}

	type Packet struct {
		OptionCount uint8    `bin:"0:1"`
		Options     []Option `bin:"1:var,len:OptionCount,repeat"`
	}

	pkt := Packet{
		OptionCount: 2,
		Options: []Option{
			{Delta: 11, Length: 4, Value: []byte("host")},
			{Delta: 15, Length: 4, Value: []byte("path")},
		},
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.OptionCount != pkt.OptionCount {
		t.Errorf("OptionCount = %d, want %d", decoded.OptionCount, pkt.OptionCount)
	}
	if len(decoded.Options) != len(pkt.Options) {
		t.Errorf("len(Options) = %d, want %d", len(decoded.Options), len(pkt.Options))
	}
	for i := range pkt.Options {
		if decoded.Options[i].Delta != pkt.Options[i].Delta {
			t.Errorf("Options[%d].Delta = %d, want %d", i, decoded.Options[i].Delta, pkt.Options[i].Delta)
		}
		if decoded.Options[i].Length != pkt.Options[i].Length {
			t.Errorf("Options[%d].Length = %d, want %d", i, decoded.Options[i].Length, pkt.Options[i].Length)
		}
		if !reflect.DeepEqual(decoded.Options[i].Value, pkt.Options[i].Value) {
			t.Errorf("Options[%d].Value = %v, want %v", i, decoded.Options[i].Value, pkt.Options[i].Value)
		}
	}
}

// TestArrayField_Float 测试浮点数数组
func TestArrayField_Float(t *testing.T) {
	type Packet struct {
		Count  uint8     `bin:"0:1"`
		Values []float32 `bin:"1:var,len:Count,repeat,size:4:be"`
	}

	pkt := Packet{
		Count:  3,
		Values: []float32{1.5, 2.5, 3.5},
	}

	data, err := Marshal(&pkt)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Packet
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.Count != pkt.Count {
		t.Errorf("Count = %d, want %d", decoded.Count, pkt.Count)
	}
	if len(decoded.Values) != len(pkt.Values) {
		t.Errorf("len(Values) = %d, want %d", len(decoded.Values), len(pkt.Values))
	}
	for i := range pkt.Values {
		if decoded.Values[i] != pkt.Values[i] {
			t.Errorf("Values[%d] = %f, want %f", i, decoded.Values[i], pkt.Values[i])
		}
	}
}

// TestArrayField_Errors 测试数组字段错误处理
func TestArrayField_Errors(t *testing.T) {
	t.Run("缺少len选项", func(t *testing.T) {
		type Packet struct {
			Values []uint16 `bin:"0:var,repeat,size:2:be"`
		}
		_, err := CompileCodec(reflect.TypeOf(Packet{}))
		if err == nil {
			t.Error("expected error for missing len option")
		}
	})

	t.Run("数据不足", func(t *testing.T) {
		type Packet struct {
			Count  uint8    `bin:"0:1"`
			Values []uint16 `bin:"1:var,len:Count,repeat,size:2:be"`
		}
		// Count=3 但只有2个元素的数据
		data := []byte{0x03, 0x00, 0x64, 0x00, 0xC8}
		var pkt Packet
		err := Unmarshal(data, &pkt)
		if err == nil {
			t.Error("expected error for insufficient data")
		}
	})
}

// BenchmarkArrayField 基准测试
func BenchmarkArrayField(b *testing.B) {
	type Packet struct {
		Count  uint8    `bin:"0:1"`
		Values []uint16 `bin:"1:var,len:Count,repeat,size:2:be"`
	}

	pkt := Packet{
		Count:  10,
		Values: make([]uint16, 10),
	}
	for i := range pkt.Values {
		pkt.Values[i] = uint16(i * 100)
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Marshal(&pkt)
		}
	})

	data, _ := Marshal(&pkt)
	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var decoded Packet
			_ = Unmarshal(data, &decoded)
		}
	})
}
