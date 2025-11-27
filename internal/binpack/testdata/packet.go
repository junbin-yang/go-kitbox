package testdata

type Packet struct {
	Magic  uint32 `bin:"0:4:be"`
	Type   uint8  `bin:"4:1"`
	Length uint16 `bin:"5:2:le"`
}
