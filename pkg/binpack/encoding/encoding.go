package encoding

import (
	"encoding/hex"
)

// Encoding 字符串编码接口
type Encoding interface {
	Encode(s string) []byte
	Decode(b []byte) string
}

// utf8Encoding UTF-8 编码（默认）
type utf8Encoding struct{}

func (e *utf8Encoding) Encode(s string) []byte {
	return []byte(s)
}

func (e *utf8Encoding) Decode(b []byte) string {
	return string(b)
}

// asciiEncoding ASCII 编码
type asciiEncoding struct{}

func (e *asciiEncoding) Encode(s string) []byte {
	return []byte(s)
}

func (e *asciiEncoding) Decode(b []byte) string {
	return string(b)
}

// hexEncoding 十六进制编码
type hexEncoding struct{}

func (e *hexEncoding) Encode(s string) []byte {
	dst := make([]byte, hex.EncodedLen(len(s)))
	hex.Encode(dst, []byte(s))
	return dst
}

func (e *hexEncoding) Decode(b []byte) string {
	dst := make([]byte, hex.DecodedLen(len(b)))
	n, _ := hex.Decode(dst, b)
	return string(dst[:n])
}

// 预定义的编码器
var (
	UTF8Encoding  Encoding = &utf8Encoding{}
	ASCIIEncoding Encoding = &asciiEncoding{}
	HexEncoding   Encoding = &hexEncoding{}
)

// GetEncoding 根据名称获取编码器
func GetEncoding(name string) Encoding {
	switch name {
	case "utf8", "":
		return UTF8Encoding
	case "ascii":
		return ASCIIEncoding
	case "hex":
		return HexEncoding
	default:
		return UTF8Encoding
	}
}
