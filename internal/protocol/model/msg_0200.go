package model

import (
	"github.com/fakeyanss/jt808-server-go/internal/codec/hex"
	"github.com/rs/zerolog/log"
)

//ADAS：高级驾驶辅助系统 (Advanced Driver Assistant System)
//DSM：驾驶员状态监测 (Driving State Monitoring)
//TPMS：轮胎气压监测系统（Tire Pressure Monitoring Systems）
//BSD:盲点监测（Blind Spot Detection）
//CAN：控制器局域网络（Controller Area Network）

const (
	ExtraIdAiADAS = 0x64
	ExtraIdAiDSM  = 0x65
	ExtraIdAiTPMS = 0x66
	ExtraIdAiBSD  = 0x67
)

// 0x01:疲劳驾驶报警
// 0x02:接打电话报警
// 0x03:抽烟报警
// 0x04:分神驾驶报警
// 0x05:驾驶员异常报警
// 0x06~0x0F：用户自定义
// 0x10：自动抓拍事件
// 0x11：驾驶员变更事件
// 0x12~0x1F：用户自定义
const (
	AlarmDSMFatigue           = 0x01
	AlarmDSMCallPhone         = 0x02
	AlarmDSMSmoking           = 0x03
	AlarmDSMDistractedDriving = 0x04
	AlarmDSMDriverAbnormal    = 0x05
	AlarmDSMAutoCapture       = 0x10
	AlarmDSMDriverChange      = 0x011
)

const (
	OFF = false
	ON  = true
)

// ExtraMsg
// 附加消息
//type ExtraMsg struct {
//	// 扩展ID，
//	//0x64 高级驾驶辅助系统报警信息，定义见表 4-15
//	//0x65 驾驶员状态监测系统报警信息，定义见表 4-17
//	//0x66 胎压监测系统报警信息，定义见表 4-18
//	//0x67 盲区监测系统报警信息，定义见表 4-2
//	ID     uint8 `json:"id"`
//	Length uint8 `json:"length"`
//
//	// 当ID为adas/dsm/tpms/bsd 时
//	// Value为 AlarmMsg
//	Value any `json:"value"`
//}

// AlarmMsg
// 告警消息
type AlarmMsg struct {

	//报警ID
	//按照报警先后，从 0 开始循环累加，不区分报警类型
	ID uint32 `json:"id"`

	//标志状态
	//0x00：不可用
	//0x01：开始标志
	//0x02：结束标志
	//该字段仅适用于有开始和结束标志类型的报警或事件，
	//报警类型或事件类型无开始和结束标志，则该位不可
	//用，填入 0x00 即可
	State uint8 `json:"state"`

	//报警标识号
	// BYTE[16]
	// 起始字节  	字段名  		数据长度  	描述
	// 0     	终端ID		BYTE[7]    	7 个字节，由大写字母和数字组成
	// 7     	时间   		BCD[6]		YY-MM-DD-hh-mm-ss （GMT+8 时间）
	// 13    	序号   		BYTE		同一时间点报警的序号，从 0 循环累加
	// 14    	附件数量 		BYTE		表示该报警对应的附件数量
	// 15    	预留    		BYTE		保留
	// e.g.
	// DEVID00 202306021600 01 00 00
	//
	SeralNumber string `json:"sn"`

	// 报警详情
	// 根据category实例化数据
	// AlarmMsgADAS
	// AlarmMsgDSM
	// AlarmMsgTPMS
	// AlarmMsgBSD
	Detail any `json:"detail"`
}

// AlarmMsgDSM
// DSM - Driving State Monitoring
// 设备上报的DSM告警信息
type AlarmMsgDSM struct {

	//报警/事件类型
	//0x01:疲劳驾驶报警
	//0x02:接打电话报警
	//0x03:抽烟报警
	//0x04:分神驾驶报警
	//0x05:驾驶员异常报警
	//0x06~0x0F：用户自定义
	//0x10：自动抓拍事件
	//0x11：驾驶员变更事件
	//0x12~0x1F：用户自定义
	Type uint8 `json:"type"`

	//报警级别
	//0x01：一级报警
	//0x02：二级报警
	Level uint8 `json:"level"`

	//疲劳程度
	//范围 1~10。数值越大表示疲劳程度越严重，仅在报警
	//类型为 0x01 时有效
	FatigueLevel uint8 `json:"fatigue"`

	//预留
	Reserve []uint8 `json:"reserve"`

	//车速
	//单位 Km/h。范围 0~250
	Speed uint8 `json:"speed"`

	//海拔
	//海拔高度，单位为米（m）
	Altitude uint16 `json:"altitude"`

	//纬度
	//以度为单位的纬度值乘以 10 的 6 次方，精确到百万分之一度
	Latitude uint32 `json:"latitude"`

	//经度
	// 以度为单位的纬度值乘以 10 的 6 次方，精确到百万分之一度
	Longitude uint32 `json:"longitude"`
	//日期时间
	//YY-MM-DD-hh-mm-ss （GMT+8 时间）
	Time string `json:"time"`

	//车辆状态
	//按位表示车辆其他状态：
	//Bit0 ACC 状态， 0：关闭，1：打开
	//Bit1 左转向状态，0：关闭，1：打开
	//Bit2 右转向状态， 0：关闭，1：打开
	//Bit3 雨刮器状态， 0：关闭，1：打开
	//Bit4 制动状态，0：未制动，1：制动
	//Bit5 插卡状态，0：未插卡，1：已插卡
	//Bit6~Bit9 自定义
	//Bit10 定位状态，0：未定位，1：已定位
	//Bit11~bit15 自定义
	CarState struct {
		ACC        bool `json:"acc"`
		LeftLight  bool `json:"leftLight"`
		RightLight bool `json:"rightLight"`
		Wiper      bool `json:"wiper"`
		Park       bool `json:"park"`
		Card       bool `json:"card"`
		Locate     bool `json:"locate"`
	} `json:"carState"`
}

// 附加信息
type Msg0200Extra struct {
	Id     uint8 `json:"id"`
	Length uint8 `json:"length"`
	Value  any   `json:"value"`
}

// Msg0200
// 位置信息汇报
type Msg0200 struct {
	Header     *MsgHeader     `json:"header"`
	AlarmSign  uint32         `json:"alarmSign"`  // 报警标志位
	StatusSign uint32         `json:"statusSign"` // 状态标志位
	Latitude   uint32         `json:"latitude"`   // 纬度，以度为单位的纬度值乘以10的6次方，精确到百万分之一度
	Longitude  uint32         `json:"longitude"`  // 精度，以度为单位的经度值乘以10的6次方，精确到百万分之一度
	Altitude   uint16         `json:"altitude"`   // 高程，海拔高度，单位为米(m)
	Speed      uint16         `json:"speed"`      // 速度，单位为0.1公里每小时(1/10km/h)
	Direction  uint16         `json:"direction"`  // 方向，0-359，正北为 0，顺时针
	Time       string         `json:"time"`       // YY-MM-DD-hh-mm-ss(GMT+8 时间)
	Extra      []Msg0200Extra `json:"extra"`      // 附加参数
}

type ExtraProcFunc func([]byte) any

const (
	MaxExtraFunctions uint32 = 65536
)

var (
	extraDecodeFunctions [MaxExtraFunctions]ExtraProcFunc
	extraEncodeFunctions [MaxExtraFunctions]ExtraProcFunc
)

func init() {
	//0x64 高级驾驶辅助系统报警信息，定义见表 4-15
	//0x65 驾驶员状态监测系统报警信息，定义见表 4-17
	//0x66 胎压监测系统报警信息，定义见表 4-18
	//0x67 盲区监测系统报警信息，定义见表 4-2
	extraDecodeFunctions[ExtraIdAiDSM] = extraAiDSMDecode
	extraEncodeFunctions[ExtraIdAiDSM] = extraAiDSMEncode
}

func bv(pos uint8) uint16 {
	const BASE = 0x01
	return BASE << pos
}

func cond(c bool, t, f any) any {
	if c {
		return t
	}

	return f
}

func extraAiDSMDecode(data []byte) any {
	if len(data) < 47 {
		// 不合法的数据
		// 根据协议规范，
		// dsm告警 需要47字节数据
		return nil
	}

	idx := 0
	alarm := &AlarmMsg{
		ID:    hex.ReadDoubleWord(data, &idx),
		State: hex.ReadByte(data, &idx),
		//SeralNumber: "xxx",
	}

	dsm := &AlarmMsgDSM{
		Type:         hex.ReadByte(data, &idx),
		Level:        hex.ReadByte(data, &idx),
		FatigueLevel: hex.ReadByte(data, &idx),
		Reserve:      hex.ReadBytes(data, &idx, 4),
		Speed:        hex.ReadByte(data, &idx),
		Altitude:     hex.ReadWord(data, &idx),
		Latitude:     hex.ReadDoubleWord(data, &idx),
		Longitude:    hex.ReadDoubleWord(data, &idx),
		Time:         hex.ReadBCD(data, &idx, 6),
		//CarState:     hex.ReadByte(data, &idx),
	}

	state := hex.ReadWord(data, &idx)
	dsm.CarState.ACC = cond((state&bv(0)) != 0, ON, OFF).(bool)
	dsm.CarState.LeftLight = cond((state&bv(1)) != 0, ON, OFF).(bool)
	dsm.CarState.RightLight = cond((state&bv(2)) != 0, ON, OFF).(bool)
	dsm.CarState.Wiper = cond((state&bv(3)) != 0, ON, OFF).(bool)
	dsm.CarState.Park = cond((state&bv(4)) != 0, ON, OFF).(bool)
	dsm.CarState.Card = cond((state&bv(5)) != 0, ON, OFF).(bool)
	dsm.CarState.Locate = cond((state&bv(10)) != 0, ON, OFF).(bool)

	alarm.Detail = dsm
	alarm.SeralNumber = hex.Byte2Str(hex.ReadBytes(data, &idx, 16))

	return alarm
}

func extraAiDSMEncode(data []byte) any {
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
		extra := Msg0200Extra{
			Id:     id,
			Length: length,
			//Value: hex.ReadBytes(pkt, &idx, int(length)),
		}
		fn := extraDecodeFunctions[id]
		if fn != nil {
			// 通过回调解析
			data := hex.ReadBytes(pkt, &idx, int(length))
			value := fn(data)
			if value == nil {
				// 当value为空，说明传入的参数不合法
				// 无法解析，就直接跳过
				log.Error().
					Uint8("id", id).
					Uint8("length", length).
					Str("hex", hex.Byte2Str(data)).
					Msg("Decode 0x0200 msg extra failed.")
				continue
			}
			extra.Value = value
		} else {
			// 没有处理回调，
			// 所以直接拿hex-buf
			extra.Value = hex.ReadBytes(pkt, &idx, int(length))
		}
		m.Extra = append(m.Extra, extra)
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

	for i := 0; i < len(m.Extra); i++ {
		extra := m.Extra[i]
		pkt = hex.WriteByte(pkt, extra.Id)
		pkt = hex.WriteByte(pkt, extra.Length)
		pkt = hex.WriteBytes(pkt, extra.Value.([]byte))
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
