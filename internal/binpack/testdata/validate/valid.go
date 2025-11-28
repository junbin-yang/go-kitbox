package validate

type ValidPacket struct {
	Length uint16 `bin:"0:2:be"`
	Type   uint8  `bin:"2:1"`
	Data   []byte `bin:"3:var,len:Length"`
}
