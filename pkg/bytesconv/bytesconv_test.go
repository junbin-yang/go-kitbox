package bytesconv

import (
	"testing"
)

func TestStringToBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple string", "hello"},
		{"unicode string", "你好世界"},
		{"special chars", "!@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToBytes(tt.input)
			if string(result) != tt.input {
				t.Errorf("StringToBytes(%q) = %q, want %q", tt.input, string(result), tt.input)
			}
		})
	}
}

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty bytes", []byte{}, ""},
		{"simple bytes", []byte("hello"), "hello"},
		{"unicode bytes", []byte("你好世界"), "你好世界"},
		{"special chars", []byte("!@#$%^&*()"), "!@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToString(tt.input)
			if result != tt.want {
				t.Errorf("BytesToString(%v) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []string{
		"",
		"hello world",
		"你好世界",
		"mixed 混合 content",
		"1234567890",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			bytes := StringToBytes(tt)
			result := BytesToString(bytes)
			if result != tt {
				t.Errorf("round trip failed: got %q, want %q", result, tt)
			}
		})
	}
}

func BenchmarkStringToBytes(b *testing.B) {
	s := "hello world benchmark test string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StringToBytes(s)
	}
}

func BenchmarkBytesToString(b *testing.B) {
	bytes := []byte("hello world benchmark test string")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BytesToString(bytes)
	}
}

func BenchmarkStandardStringToBytes(b *testing.B) {
	s := "hello world benchmark test string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = []byte(s)
	}
}

func BenchmarkStandardBytesToString(b *testing.B) {
	bytes := []byte("hello world benchmark test string")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string(bytes)
	}
}
