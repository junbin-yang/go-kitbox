package validate

type InvalidPacket struct {
	Field1 uint16 `bin:"0:2:be"`
	Field2 uint8  `bin:"0:1"` // 偏移量冲突
}

type MissingLenPacket struct {
	Type uint8  `bin:"0:1"`
	Data []byte `bin:"1:var,len:Length"` // Length 字段不存在
}
