package binpack

import (
	"errors"
	"testing"
)

// TestDetailedDecodeError 测试详细解码错误的实际使用
func TestDetailedDecodeError(t *testing.T) {
	type Packet struct {
		Magic   uint32   `bin:"0:4:be"`
		Count   uint8    `bin:"4:1"`
		Values  []uint16 `bin:"5:var,len:Count,repeat,size:2:be"`
		Payload []byte   `bin:"100:var,len:Count"`
	}

	tests := []struct {
		name          string
		data          []byte
		wantFieldName string
		wantOffset    int
		wantExpected  int
		wantActual    int
	}{
		{
			name:          "数据太短-基础字段",
			data:          []byte{0x12, 0x34}, // 只有2字节，期望至少5字节
			wantFieldName: "",
			wantOffset:    0,
			wantExpected:  5,
			wantActual:    2,
		},
		{
			name: "数组字段数据不足",
			data: []byte{
				0x12, 0x34, 0x56, 0x78, // Magic
				0x03,       // Count=3
				0x00, 0x64, // 第1个值
				0x00,       // 第2个值不完整
			},
			wantFieldName: "Values",
			wantOffset:    5,
			wantExpected:  6, // 3个uint16 = 6字节
			wantActual:    3, // 实际只有3字节
		},
		{
			name: "变长字段数据不足",
			data: []byte{
				0x12, 0x34, 0x56, 0x78, // Magic
				0x02, // Count=2
				0x00, 0x64, // Values[0]
				0x00, 0xC8, // Values[1]
				// Payload应该在offset 100，但数据只有9字节
			},
			wantFieldName: "Payload",
			wantOffset:    100,
			wantExpected:  2,
			wantActual:    -91, // len(data)-offset = 9-100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pkt Packet
			err := Unmarshal(tt.data, &pkt)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// 检查是否是 DecodeError
			var decErr *DecodeError
			if !errors.As(err, &decErr) {
				t.Fatalf("expected DecodeError, got %T: %v", err, err)
			}

			// 验证错误详情
			if tt.wantFieldName != "" && decErr.FieldName != tt.wantFieldName {
				t.Errorf("FieldName = %q, want %q", decErr.FieldName, tt.wantFieldName)
			}
			if decErr.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", decErr.Offset, tt.wantOffset)
			}
			if decErr.ExpectedSize != tt.wantExpected {
				t.Errorf("ExpectedSize = %d, want %d", decErr.ExpectedSize, tt.wantExpected)
			}
			if decErr.ActualSize != tt.wantActual {
				t.Errorf("ActualSize = %d, want %d", decErr.ActualSize, tt.wantActual)
			}

			t.Logf("详细错误信息: %v", err)
		})
	}
}

// TestDetailedErrorUsage 演示如何在实际代码中使用详细错误
func TestDetailedErrorUsage(t *testing.T) {
	type Message struct {
		Type    uint8    `bin:"0:1"`
		Length  uint16   `bin:"1:2:le"`
		Payload []byte   `bin:"3:var,len:Length"`
	}

	// 模拟接收到的不完整数据
	incompleteData := []byte{0x01, 0x0A, 0x00, 0x48, 0x65, 0x6C} // Type=1, Length=10, 但只有3字节payload

	var msg Message
	err := Unmarshal(incompleteData, &msg)
	if err != nil {
		// 方式1: 直接打印错误（包含所有详细信息）
		t.Logf("解码失败: %v", err)

		// 方式2: 使用类型断言获取详细信息
		var decErr *DecodeError
		if errors.As(err, &decErr) {
			t.Logf("字段: %s", decErr.FieldName)
			t.Logf("类型: %s", decErr.FieldType)
			t.Logf("偏移: %d", decErr.Offset)
			t.Logf("期望长度: %d 字节", decErr.ExpectedSize)
			t.Logf("实际长度: %d 字节", decErr.ActualSize)
			t.Logf("错误描述: %s", decErr.Message)

			// 可以根据错误信息做特定处理
			if decErr.ActualSize < decErr.ExpectedSize {
				t.Logf("需要继续接收 %d 字节", decErr.ExpectedSize-decErr.ActualSize)
			}
		}

		// 方式3: 检查是否是特定字段的错误
		if errors.As(err, &decErr) && decErr.FieldName == "Payload" {
			t.Log("Payload字段解码失败，可能是网络传输不完整")
		}
	}
}

// BenchmarkDetailedError 基准测试详细错误的性能影响
func BenchmarkDetailedError(b *testing.B) {
	type Packet struct {
		Magic  uint32 `bin:"0:4:be"`
		Length uint16 `bin:"4:2:le"`
		Data   []byte `bin:"6:var,len:Length"`
	}

	// 正常数据
	validData := []byte{0x12, 0x34, 0x56, 0x78, 0x05, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05}

	b.Run("成功解码", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var pkt Packet
			_ = Unmarshal(validData, &pkt)
		}
	})

	// 错误数据
	invalidData := []byte{0x12, 0x34, 0x56, 0x78, 0x0A, 0x00, 0x01, 0x02} // Length=10但只有2字节

	b.Run("解码错误", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var pkt Packet
			_ = Unmarshal(invalidData, &pkt)
		}
	})
}
