package model

import (
	"github.com/fakeyanss/jt808-server-go/internal/codec/hex"
)

type Msg0200Expand struct {
	Id     uint8 `json:"id"`
	Length uint8 `json:"length"`
	//Payload []byte `json:"payload"`
	Payload any `json:"payload"`
}

// 位置信息汇报
type Msg0200 struct {
	Header     *MsgHeader      `json:"header"`
	AlarmSign  uint32          `json:"alarmSign"`  // 报警标志位
	StatusSign uint32          `json:"statusSign"` // 状态标志位
	Latitude   uint32          `json:"latitude"`   // 纬度，以度为单位的纬度值乘以10的6次方，精确到百万分之一度
	Longitude  uint32          `json:"longitude"`  // 精度，以度为单位的经度值乘以10的6次方，精确到百万分之一度
	Altitude   uint16          `json:"altitude"`   // 高程，海拔高度，单位为米(m)
	Speed      uint16          `json:"speed"`      // 速度，单位为0.1公里每小时(1/10km/h)
	Direction  uint16          `json:"direction"`  // 方向，0-359，正北为 0，顺时针
	Time       string          `json:"time"`       // YY-MM-DD-hh-mm-ss(GMT+8 时间)
	Expand     []Msg0200Expand `json:"expand"`     // 附加参数
}

type ExpandProcFunc func([]byte) any

const (
	MaxExpandFunctions uint32 = 65536
)

var (
	expandDecodeFunctions [MaxExpandFunctions]ExpandProcFunc
	expandEncodeFunctions [MaxExpandFunctions]ExpandProcFunc
)

func init() {
	expandDecodeFunctions[0x65] = expandAiDSMDecode
	expandEncodeFunctions[0x65] = expandAiDSMEncode
}

func expandAiDSMDecode(data []byte) any {
	return data
}

func expandAiDSMEncode(data []byte) any {
	return data
}

func (m *Msg0200) Decode(packet *PacketData) error {
	m.Header = packet.Header
	pkt, idx := packet.Body, 0
	m.AlarmSign = hex.ReadDoubleWord(pkt, &idx)
	m.StatusSign = hex.ReadDoubleWord(pkt, &idx)
	m.Latitude = hex.ReadDoubleWord(pkt, &idx)
	m.Longitude = hex.ReadDoubleWord(pkt, &idx)
	m.Altitude = hex.ReadWord(pkt, &idx)
	m.Speed = hex.ReadWord(pkt, &idx)
	m.Direction = hex.ReadWord(pkt, &idx)
	m.Time = hex.ReadBCD(pkt, &idx, 6)

	for idx < len(pkt) {
		id := hex.ReadByte(pkt, &idx)
		length := hex.ReadByte(pkt, &idx)
		if id == 0 || (idx+int(length)) > int(m.Header.Attr.BodyLength) {
			// id 无效
			// 或者 数据已经越界，不要再继续解码了
			break
		}
		expand := Msg0200Expand{
			Id:     id,
			Length: length,
			//Payload: hex.ReadBytes(pkt, &idx, int(length)),
		}
		fn := expandDecodeFunctions[id]
		if fn != nil {
			// 通过回调解析
			expand.Payload = fn(hex.ReadBytes(pkt, &idx, int(length)))
		} else {
			// 没有处理回调，
			// 所以直接拿hex-buf
			expand.Payload = hex.ReadBytes(pkt, &idx, int(length))
		}
		m.Expand = append(m.Expand, expand)
	}
	return nil
}

func (m *Msg0200) Encode() (pkt []byte, err error) {
	pkt = hex.WriteDoubleWord(pkt, m.AlarmSign)
	pkt = hex.WriteDoubleWord(pkt, m.StatusSign)
	pkt = hex.WriteDoubleWord(pkt, m.Latitude)
	pkt = hex.WriteDoubleWord(pkt, m.Longitude)
	pkt = hex.WriteWord(pkt, m.Altitude)
	pkt = hex.WriteWord(pkt, m.Speed)
	pkt = hex.WriteWord(pkt, m.Direction)
	pkt = hex.WriteBCD(pkt, m.Time)

	for i := 0; i < len(m.Expand); i++ {
		expand := m.Expand[i]
		pkt = hex.WriteByte(pkt, expand.Id)
		pkt = hex.WriteByte(pkt, expand.Length)
		pkt = hex.WriteBytes(pkt, expand.Payload.([]byte))
	}

	pkt, err = writeHeader(m, pkt)
	return pkt, err
}

func (m *Msg0200) GetHeader() *MsgHeader {
	return m.Header
}

func (m *Msg0200) GenOutgoing(_ JT808Msg) error {
	return nil
}
