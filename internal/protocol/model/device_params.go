package model

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"reflect"
	"sort"

	"github.com/fakeyanss/jt808-server-go/internal/codec/hex"
)

var (
	ErrDecodeDeviceParams   = errors.New("Fail to decode device params")
	ErrEncodeDeviceParams   = errors.New("Fail to encode device params")
	ErrParamIDNotSupportted = errors.New("Param id is not supportted")
)

type DeviceParams struct {
	DevicePhone string       `json:"-"`        // 关联device phone
	ParamCnt    uint8        `json:"total"`    // 参数项个数
	Params      []*ParamData `json:"settings"` // 参数项列表
}

func (p *DeviceParams) Decode(phone string, cnt uint8, pkt []byte) error {
	p.DevicePhone = phone
	p.ParamCnt = cnt
	idx := 0
	for i := 0; i < int(cnt); i++ {
		param := &ParamData{}
		err := param.Decode(pkt, &idx)
		if err != nil {
			return err
		}

		p.Params = append(p.Params, param)
	}
	return nil
}

func (p *DeviceParams) Encode() (pkt []byte, err error) {
	p.ParamCnt = uint8(len(p.Params))
	pkt = hex.WriteByte(pkt, p.ParamCnt)
	for _, arg := range p.Params {
		paramBytes, err := arg.Encode()
		if err != nil {
			// skip this err
			log.Error().Err(err).Str("device", p.DevicePhone).Msg("Fail to encode device param")
			continue
		}
		pkt = hex.WriteBytes(pkt, paramBytes)
	}
	return pkt, nil
}

func (p *DeviceParams) Update(newParams *DeviceParams) {
	paramMap := make(map[uint32]*ParamData)
	for _, param := range p.Params {
		paramMap[param.ParamID] = param
	}
	for _, newParam := range newParams.Params {
		id := newParam.ParamID
		if _, ok := paramMap[id]; ok {
			paramMap[id] = newParam
		}
	}
	mergeParams := []*ParamData{}
	for _, param := range paramMap {
		mergeParams = append(mergeParams, param)
	}
	sort.Slice(mergeParams, func(i, j int) bool {
		return mergeParams[i].ParamID < mergeParams[j].ParamID
	})
	p.Params = mergeParams
	p.ParamCnt = uint8(len(mergeParams))
}

type ParamData struct {
	ParamID  uint32 `json:"id"`     // 参数ID
	ParamLen uint8  `json:"length"` // 参数长度

	// SettingsAiDSM
	ParamValue any    `json:"value"` // 参数值
	ParamDesc  string `json:"desc"`  // 参数说明
}

func (p *ParamData) Decode(pkt []byte, idx *int) error {
	p.ParamID = hex.ReadDoubleWord(pkt, idx)
	p.ParamLen = hex.ReadByte(pkt, idx)
	fn, ok := argTable[p.ParamID]
	if !ok {
		log.Warn().Str("ParamID", fmt.Sprintf("0x%04x", p.ParamID)).Err(ErrParamIDNotSupportted).Msg("skip it")
	}
	p.ParamValue = fn.decode(pkt, idx, int(p.ParamLen))
	return nil
}

func (p *ParamData) Encode() (pkt []byte, err error) {
	pkt = hex.WriteDoubleWord(pkt, p.ParamID)
	if fn, ok := argTable[p.ParamID]; ok {
		value := fn.encode(p.ParamValue)
		pkt = hex.WriteByte(pkt, uint8(len(value)))
		pkt = hex.WriteBytes(pkt, value)
		return pkt, nil
	}
	log.Warn().
		Str("ParamID", fmt.Sprintf("0x%04x", p.ParamID)).
		Err(ErrParamIDNotSupportted).
		Msg("skip it")
	return nil, ErrParamIDNotSupportted
}

type paramFn struct {
	decode func([]byte, *int, int) any
	encode func(any) (pkt []byte)
}

// !!!特别注意，any类型被encoding/json Unmarshal后，会转为默认的类型，如下:
//
//	bool, for JSON booleans
//	float64, for JSON numbers
//	string, for JSON strings
//	[]interface{}, for JSON arrays
//	map[string]interface{}, for JSON objects
//	nil for JSON null
//
// 所以需要再encode时，将其按照json默认类型推断，再进行强转

func any2uint8(a any) uint8 {
	if b, ok := a.(float64); ok {
		return uint8(b)
	}
	return a.(uint8)
}

func writeByteAny(pkt []byte, num any) []byte {
	return hex.WriteByte(pkt, any2uint8(num))
}

func any2uint16(a any) uint16 {
	if b, ok := a.(float64); ok {
		return uint16(b)
	}
	return a.(uint16)
}

func writeWordAny(pkt []byte, num any) []byte {
	return hex.WriteWord(pkt, any2uint16(num))
}

func any2uint32(a any) uint32 {
	if b, ok := a.(float64); ok {
		return uint32(b)
	}
	return a.(uint32)
}

func writeDoubleWordAny(pkt []byte, num any) []byte {
	return hex.WriteDoubleWord(pkt, any2uint32(num))
}

// []byte -> struct
func decodeDsm(b []byte, idx *int, paramLen int) any {
	dsm := &SettingsAiDSM{}

	// 遍历结构体
	t := reflect.TypeOf(*dsm)
	v := reflect.ValueOf(dsm).Elem()
	for k := 0; k < t.NumField(); k++ {
		field := v.Field(k)
		if !field.CanSet() {
			continue
		}
		switch field.Type().Size() {
		case 1:
			field.Set(reflect.ValueOf(hex.ReadByte(b, idx)))
			break
		case 2:
			field.Set(reflect.ValueOf(hex.ReadWord(b, idx)))
			break
		case 4:
			field.Set(reflect.ValueOf(hex.ReadDoubleWord(b, idx)))
			break
		}
	}
	return &dsm
}

// struct -> []byte
func encodeDsm(a any) (pkt []byte) {
	data, _ := json.Marshal(a)

	dsm := &SettingsAiDSM{}
	err := json.Unmarshal(data, dsm)
	if err != nil {
		return
	}

	// 遍历结构体
	t := reflect.TypeOf(*dsm)
	v := reflect.ValueOf(*dsm)
	for k := 0; k < t.NumField(); k++ {
		field := v.Field(k)
		value := field.Interface()

		switch field.Type().Size() {
		case 1:
			pkt = writeByteAny(pkt, value)
			break
		case 2:
			pkt = writeWordAny(pkt, value)
			break
		case 4:
			pkt = writeDoubleWordAny(pkt, value)
			break
		}
	}

	return
}

var (
	decodeByte       = func(b []byte, idx *int, paramLen int) any { return hex.ReadByte(b, idx) }
	encodeByte       = func(a any) (pkt []byte) { return writeByteAny(pkt, a) }
	decodeWord       = func(b []byte, idx *int, paramLen int) any { return hex.ReadWord(b, idx) }
	encodeWord       = func(a any) (pkt []byte) { return writeWordAny(pkt, a) }
	decodeDoubleWord = func(b []byte, idx *int, paramLen int) any { return hex.ReadDoubleWord(b, idx) }
	encodeDoubleWord = func(a any) (pkt []byte) { return writeDoubleWordAny(pkt, a) }
	decodeBytes      = func(b []byte, idx *int, paramLen int) any { return hex.ReadBCD(b, idx, paramLen) } // transform bytes to string
	encodeBytes      = func(a any) (pkt []byte) { return hex.WriteBCD(pkt, a.(string)) }                   // transform string to bytes
	decodeString     = func(b []byte, idx *int, paramLen int) any { return hex.ReadString(b, idx, paramLen) }
	encodeString     = func(a any) (pkt []byte) { return hex.WriteString(pkt, a.(string)) }
	decodeGBK        = func(b []byte, idx *int, paramLen int) any { return hex.ReadGBK(b, idx, paramLen) }
	encodeGBK        = func(a any) (pkt []byte) { return hex.WriteGBK(pkt, a.(string)) }
)

var argTable = map[uint32]*paramFn{
	// JT808 param

	// 终端心跳发送间隔,单位为秒(s)
	0x0001: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// TCP消息应答超时时间,单位为秒(s)
	0x0002: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// TCP消息重传次数
	0x0003: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// UDP消息应答超时时间,单位为秒(s)
	0x0004: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// UDP消息重传次数
	0x0005: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// SMS消息应答超时时间,单位为秒(s)
	0x0006: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// SMS消息重传次数
	0x0007: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 主服务器APN,无线通信拨号访问点.若网络制式为CDMA,则该处为PPP拨号号码
	0x0010: {decode: decodeGBK, encode: encodeGBK},
	// 主服务器无线通信拨号用户名
	0x0011: {decode: decodeGBK, encode: encodeGBK},
	// 主服务器无线通信拨号密码
	0x0012: {decode: decodeGBK, encode: encodeGBK},
	// 主服务器地址,IP或域名
	0x0013: {decode: decodeGBK, encode: encodeGBK},
	// 备份服务器APN,无线通信拨号访问点
	0x0014: {decode: decodeGBK, encode: encodeGBK},
	// 备份服务器无线通信拨号用户名
	0x0015: {decode: decodeGBK, encode: encodeGBK},
	// 备份服务器无线通信拨号密码
	0x0016: {decode: decodeGBK, encode: encodeGBK},
	// 备份服务器地址,IP或域名(2019版以冒号分割主机和端口,多个服务器使用分号分隔)
	0x0017: {decode: decodeGBK, encode: encodeGBK},
	// (JT808 2013)服务器TCP端口
	0x0018: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// (JT808 2013)服务器UDP端口
	0x0019: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 道路运输证IC卡认证主服务器IP地址或域名
	0x001A: {decode: decodeGBK, encode: encodeGBK},
	// 道路运输证IC卡认证主服务器TCP端口
	0x001B: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 道路运输证IC卡认证主服务器UDP端口
	0x001C: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 道路运输证IC卡认证主服务器IP地址或域名,端口同主服务器
	0x001D: {decode: decodeGBK, encode: encodeGBK},
	// 位置汇报策略：0.定时汇报 1.定距汇报 2.定时和定距汇报
	0x0020: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 位置汇报方案：0.根据ACC状态 1.根据登录状态和ACC状态,先判断登录状态,若登录再根据ACC状态
	0x0021: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 驾驶员未登录汇报时间间隔,单位为秒(s),>0
	0x0022: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// (JT808 2019)从服务器APN.该值为空时,终端应使用主服务器相同配置
	0x0023: {decode: decodeGBK, encode: encodeGBK},
	// (JT808 2019)从服务器无线通信拨号用户名.该值为空时,终端应使用主服务器相同配置
	0x0024: {decode: decodeGBK, encode: encodeGBK},
	// (JT808 2019)从服务器无线通信拨号密码.该值为空时,终端应使用主服务器相同配置
	0x0025: {decode: decodeGBK, encode: encodeGBK},
	// (JT808 2019)从服务器备份地址、IP或域名.主服务器IP地址或域名,端口同主服务器
	0x0026: {decode: decodeGBK, encode: encodeGBK},
	// 休眠时汇报时间间隔,单位为秒(s),>0
	0x0027: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 紧急报警时汇报时间间隔,单位为秒(s),>0
	0x0028: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 缺省时间汇报间隔,单位为秒(s),>0
	0x0029: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 缺省距离汇报间隔,单位为米(m),>0
	0x002C: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 驾驶员未登录汇报距离间隔,单位为米(m),>0
	0x002D: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 休眠时汇报距离间隔,单位为米(m),>0
	0x002E: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 紧急报警时汇报距离间隔,单位为米(m),>0
	0x002F: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 拐点补传角度,<180°
	0x0030: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 电子围栏半径,单位为米
	0x0031: {decode: decodeWord, encode: encodeWord},
	// (JT808 2019)违规行驶时段范围,精确到分。
	//   byte1：违规行驶开始时间的小时部分；
	//   byte2：违规行驶开始的分钟部分；
	//   byte3：违规行驶结束时间的小时部分；
	//   byte4：违规行驶结束时间的分钟部分。
	0x0032: {decode: decodeBytes, encode: encodeBytes},
	// 监控平台电话号码
	0x0040: {decode: decodeGBK, encode: encodeGBK},
	// 复位电话号码,可采用此电话号码拨打终端电话让终端复位
	0x0041: {decode: decodeGBK, encode: encodeGBK},
	// 恢复出厂设置电话号码,可采用此电话号码拨打终端电话让终端恢复出厂设置
	0x0042: {decode: decodeGBK, encode: encodeGBK},
	// 监控平台SMS电话号码
	0x0043: {decode: decodeGBK, encode: encodeGBK},
	// 接收终端SMS文本报警号码
	0x0044: {decode: decodeGBK, encode: encodeGBK},
	// 终端电话接听策略,0.自动接听 1.ACC ON时自动接听,OFF时手动接听
	0x0045: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 每次最长通话时间,单位为秒(s),0为不允许通话,0xFFFFFFFF为不限制
	0x0046: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 当月最长通话时间,单位为秒(s),0为不允许通话,0xFFFFFFFF为不限制
	0x0047: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 监听电话号码
	0x0048: {decode: decodeGBK, encode: encodeGBK},
	// 监管平台特权短信号码
	0x0049: {decode: decodeGBK, encode: encodeGBK},
	// 报警屏蔽字.与位置信息汇报消息中的报警标志相对应,相应位为1则相应报警被屏蔽
	0x0050: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 报警发送文本SMS开关,与位置信息汇报消息中的报警标志相对应,相应位为1则相应报警时发送文本SMS
	0x0051: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 报警拍摄开关,与位置信息汇报消息中的报警标志相对应,相应位为1则相应报警时摄像头拍摄
	0x0052: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 报警拍摄存储标志,与位置信息汇报消息中的报警标志相对应,相应位为1则对相应报警时牌的照片进行存储,否则实时长传
	0x0053: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 关键标志,与位置信息汇报消息中的报警标志相对应,相应位为1则对相应报警为关键报警
	0x0054: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 最高速度，单位为千米每小时(km/h)
	0x0055: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 超速持续时间,单位为秒(s)
	0x0056: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 连续驾驶时间门限,单位为秒(s)
	0x0057: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 当天累计驾驶时间门限,单位为秒(s)
	0x0058: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 最小休息时间,单位为秒(s)
	0x0059: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 最长停车时间,单位为秒(s)
	0x005A: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 超速预警差值
	0x005B: {decode: decodeWord, encode: encodeWord},
	// 疲劳驾驶预警插值
	0x005C: {decode: decodeWord, encode: encodeWord},
	// 碰撞报警参数
	0x005D: {decode: decodeWord, encode: encodeWord},
	// 侧翻报警参数
	0x005E: {decode: decodeWord, encode: encodeWord},
	// 定时拍照参数
	0x0064: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 定距拍照参数
	0x0065: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 视频质量,1~10,1最好
	0x0070: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 亮度,0~255
	0x0071: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 对比度,0~127
	0x0072: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 饱和度,0~127
	0x0073: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 色度,0~255
	0x0074: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 车辆里程表读数，1/10km
	0x0080: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// 车辆所在的省域ID
	0x0081: {decode: decodeWord, encode: encodeWord},
	// 车辆所在的省域ID
	0x0082: {decode: decodeWord, encode: encodeWord},
	// 公安交通管理部门颁发的机动车号牌
	0x0083: {decode: decodeGBK, encode: encodeGBK},
	// 车牌颜色，按照JT415-2006的5.4.12
	// 01-蓝色;02-黄色;03-黑色;04-白色;05-绿色;09-其他;91-农黄色;92-农绿色;93-黄绿色;94-渐变绿
	0x0084: {decode: decodeByte, encode: encodeByte},
	// GNSS定位模式，定义如下：
	//   bit0，0:禁用GPS定位，1:启用 GPS 定位;
	//   bit1，0:禁用北斗定位，1:启用北斗定位;
	//   bit2，0:禁用GLONASS 定位，1:启用GLONASS定位;
	//   bit3，0:禁用Galileo定位，1:启用Galileo定位
	0x0090: {decode: decodeByte, encode: encodeByte},
	// GNSS波特率，定义如下：
	//   0x00:4800;
	//   0x01:9600;
	//   0x02:19200;
	//   0x03:38400;
	//   0x04:57600;
	//   0x05:115200
	0x0091: {decode: decodeByte, encode: encodeByte},
	// GNSS模块详细定位数据输出频率，定义如下：
	//   0x00:500ms;
	//   0x01:1000ms(默认值);
	//   0x02:2000ms;
	//   0x03:3000ms;
	//   0x04:4000ms
	0x0092: {decode: decodeByte, encode: encodeByte},
	// GNSS模块详细定位数据采集频率，单位为秒，默认为 1。
	0x0093: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// GNSS模块详细定位数据上传方式:
	//   0x00，本地存储，不上传(默认值);
	//   0x01，按时间间隔上传;
	//   0x02，按距离间隔上传;
	//   0x0B，按累计时间上传，达到传输时间后自动停止上传;
	//   0x0C，按累计距离上传，达到距离后自动停止上传;
	//   0x0D，按累计条数上传，达到上传条数后自动停止上传。
	0x0094: {decode: decodeByte, encode: encodeByte},
	// GNSS模块详细定位数据上传设置, 关联0x0094:
	// 上传方式为 0x01 时，单位为秒;
	// 上传方式为 0x02 时，单位为米;
	// 上传方式为 0x0B 时，单位为秒;
	// 上传方式为 0x0C 时，单位为米;
	// 上传方式为 0x0D 时，单位为条。
	0x0095: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// CAN总线通道1采集时间间隔(ms)，0表示不采集
	0x0100: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// CAN总线通道1上传时间间隔(s)，0表示不上传
	0x0101: {decode: decodeWord, encode: encodeWord},
	// CAN总线通道2采集时间间隔(ms)，0表示不采集
	0x0102: {decode: decodeDoubleWord, encode: encodeDoubleWord},
	// CAN总线通道2上传时间间隔(s)，0表示不上传
	0x0103: {decode: decodeWord, encode: encodeWord},
	// CAN总线ID单独采集设置:
	//   bit63-bit32 表示此 ID 采集时间间隔(ms)，0 表示不采集;
	//   bit31 表示 CAN 通道号，0:CAN1，1:CAN2;
	//   bit30 表示帧类型，0:标准帧，1:扩展帧;
	//   bit29 表示数据采集方式，0:原始数据，1:采集区间的计算值;
	//   bit28-bit0 表示 CAN 总线 ID。
	0x0110: {decode: decodeString, encode: encodeString},

	// JT1078 param
	// 音视频参数设置
	0x0075: {decode: decodeBytes, encode: encodeBytes},
	// 音视频通道列表设置
	0x0076: {decode: decodeBytes, encode: encodeBytes},
	// 单独通道视频参数设置
	0x0077: {decode: decodeBytes, encode: encodeBytes},
	// 特殊报警录像参数设置
	0x0079: {decode: decodeBytes, encode: encodeBytes},
	// 视频相关报警屏蔽字
	0x007A: {decode: decodeBytes, encode: encodeBytes},
	// 图像分析报警参数设置
	0x007B: {decode: decodeBytes, encode: encodeBytes},
	// 终端休眠唤醒模式设置
	0x007C: {decode: decodeBytes, encode: encodeBytes},

	// ai - dsm
	0xF365: {decode: decodeDsm, encode: encodeDsm},
}

// SettingsAiDSM
// 参考苏标规范4.3.1章节设置
// ID 为 SettingsIdAiDSM
type SettingsAiDSM struct {
	// 报警判断速度阈值
	// BYTE
	// 单位 km/h，取值范围 0~60，默认值 30。表示当车速
	// 高于此阈值才使能报警功能
	// 0xFF 表示不修改此参数
	AlarmSpeedThreshold uint8 `json:"alarmSpeedThreshold"`

	// 报警提示音量
	// BYTE
	// 0~8，8 最大，0 静音，默认值 6
	// 0xFF 表示不修改参数
	AlarmVolume uint8 `json:"alarmVolume"`

	// 主动拍照策略
	// BYTE
	// 0x00：不开启
	// 0x01：定时拍照
	// 0x02：定距拍照
	// 0x03：插卡触发
	// 0x04：保留
	// 默认值 0x00，
	// 0xFF 表示不修改参数
	ProactivePhotoStrategy uint8 `json:"proactivePhotoStrategy"`

	// 主动定时拍照时间间隔
	// 单位秒，取值范围 0~60000，默认值 3600
	// 0 表示不抓拍，0xFFFF 表示不修改参数
	// 主动拍照策略为 01 时有效。
	ProactivePhotoInterval uint16 `json:"proactivePhotoInterval"`

	// 主动定距拍照距离间隔
	// WORD
	// 单位米，取值范围 0~60000，默认值 200
	// 0 表示不抓拍，0xFFFF 表示不修改参数
	// 主动拍照策略为 02 时有效。
	ProactivePhotoDistanceInterval uint16 `json:"proactivePhotoDistanceInterval"`

	// 单次主动拍照张数
	// BYTE
	// 取值范围 1-10。默认值 3，
	// 0xFF 表示不修改参数
	ProactivePhotoCount uint8 `json:"proactivePhotoCount"`

	// 单次主动拍照时间间隔
	// BYTE
	// 单位 100ms，取值范围 1~5，默认值 2，
	// 0xFF 表示不修改参数
	ProactivePhotoIntervalTime uint8 `json:"proactivePhotoIntervalTime"`

	// 拍照分辨率
	// BYTE
	// 0x01：352×288
	// 0x02：704×288
	// 0x03：704×576
	// 0x04：640×480
	// 0x05：1280×720
	// 0x06：1920×1080
	// 默认值 0x01，
	// 0xFF 表示不修改参数，
	// 该参数也适用于报警触发拍照分辨率。
	PhotoResolution uint8 `json:"photoResolution"`

	// 视频录制分辨率
	// BYTE
	// 0x01：CIF
	// 0x02：HD1
	// 0x03：D1
	// 0x04：WD1
	// 0x05：VGA
	// 0x06：720P
	// 0x07：1080P
	// 默认值 0x01
	// 0xFF 表示不修改参数
	// 该参数也适用于报警触发视频分辨率
	VideoResolution uint8 `json:"videoResolution"`

	// 报警使能
	// DWORD
	// 报警使能位 0：关闭 1：打开
	// bit0：疲劳驾驶一级报警
	// bit1：疲劳驾驶二级报警
	// bit2：接打电话一级报警
	// bit3：接打电话二级报警
	// bit4：抽烟一级报警
	// bit5：抽烟二级报警
	// bit6：分神驾驶一级报警
	// bit7：分神驾驶二级报警
	// bit8：驾驶员异常一级报警
	// bit9：驾驶员异常二级报警
	// bit10~bit29：用户自定义
	// bit30~bit31：保留
	// 默认值 0x000001FF
	// 0xFFFFFFFF 表示不修改参数
	AlarmEnabled uint32 `json:"alarmEnabled"`

	// 事件使能
	// DWORD
	// 事件使能位 0：关闭 1：打开
	// bit0：驾驶员更换事件
	// bit1：主动拍照事件
	// bit2~bit29：用户自定义
	// bit30~bit31：保留
	// 默认值 0x00000003
	// 0xFFFFFFFF 表示不修改参数
	EventEnabled uint32 `json:"eventEnabled"`

	// 吸烟报警判断时间间隔
	// WORD
	// 单位秒，取值范围 0~3600。默认值为 180。表示在此
	// 时间间隔内仅触发一次吸烟报警。
	// 0xFFFF 表示不修改此参数
	SmokingAlarmInterval uint16 `json:"smokingAlarmInterval"`

	// 接打电话报警判断时间间隔
	// WORD
	// 单位秒，取值范围 0~3600。默认值为 120。表示在此
	// 时间间隔内仅触发一次接打电话报警。
	// 0xFFFF 表示不修改此参数
	PhoneCallAlarmInterval uint16 `json:"phoneCallAlarmInterval"`

	// 预留字段
	// BYTE[3] 保留字段
	ReservedField1 uint8 `json:"reservedField1"`
	ReservedField2 uint8 `json:"reservedField2"`
	ReservedField3 uint8 `json:"reservedField3"`

	// 疲劳驾驶报警分级速度阈值
	// BYTE
	// 单位 km/h，取值范围 0~220，默认值 50。表示触发报
	// 警时车速高于阈值为二级报警，否则为一级报警
	// 0xFF 表示不修改参数
	FatigueDrivingSpeedThreshold uint8 `json:"fatigueDrivingSpeedThreshold"`

	// 疲劳驾驶报警前后视频录制时间
	// BYTE
	// 单位秒，取值范围 0-60，默认值 5
	// 0 表示不录像，0xFF 表示不修改参数
	FatigueDrivingVideoRecordingTime uint8 `json:"fatigueDrivingVideoRecordingTime"`

	// 疲劳驾驶报警拍照张数
	// BYTE
	// 取值范围 0-10，缺省值 3
	// 0 表示不抓拍，0xFF 表示不修改参数
	FatigueDrivingPhotoCount uint8 `json:"fatigueDrivingPhotoCount"`

	// 疲劳驾驶报警拍照间隔时间
	// BYTE
	// 单位 100ms， 取值范围 1~5，默认 2，
	// 0xFF 表示不修改参数
	FatigueDrivingPhotoInterval uint8 `json:"fatigueDrivingPhotoInterval"`

	// 接打电话报警分级速度阈值
	// BYTE
	// 单位 km/h，取值范围 0~220，默认值 50。表示触发报
	// 警时车速高于阈值为二级报警，否则为一级报警
	// 0xFF 表示不修改参数
	PhoneCallAlarmSpeedThreshold uint8 `json:"phoneCallAlarmSpeedThreshold"`

	// 接打电话报警前后视频录制时间
	// BYTE
	// 单位秒，取值范围 0-60，默认值 5，
	// 0 表示不录像，0xFF 表示不修改参数
	PhoneCallAlarmVideoRecordingTime uint8 `json:"phoneCallAlarmVideoRecordingTime"`

	// 接打电话报警拍驾驶员面部特征照片张数
	// BYTE
	// 取值范围 1-10，默认值 3
	// 0 表示不抓拍，0xFF 表示不修改参数
	PhoneCallDriverFacePhotoCount uint8 `json:"phoneCallDriverFacePhotoCount"`

	// 接打电话报警拍驾驶员面部特征照片间隔时间
	// BYTE
	// 单位 100ms， 取值范围 1~5，默认值 2
	// 0xFF 表示不修改参数
	PhoneCallDriverFaceFeatureInterval uint8 `json:"phoneCallDriverFaceFeatureInterval"`

	// 抽烟报警分级车速阈值
	// BYTE
	// 单位 km/h，取值范围 0~220，默认值 50。表示触发报
	// 警时车速高于阈值为二级报警，否则为一级报警
	// 0xFF 表示不修改参数
	SmokingAlarmSpeedThreshold uint8 `json:"smokingAlarmSpeedThreshold"`

	// 抽烟报警前后视频录制时间
	// BYTE
	// 单位秒，取值范围 0-60，默认值 5
	// 0 表示不录像，0xFF 表示不修改参数
	SmokingAlarmVideoRecordingTime uint8 `json:"smokingAlarmVideoRecordingTime"`

	// 抽烟报警拍驾驶员面部特征照片张数
	// BYTE
	// 取值范围 1-10，默认值 3
	// 0 表示不抓拍，0xFF 表示不修改参数
	SmokingAlarmDriverFacePhotoCount uint8 `json:"smokingAlarmDriverFacePhotoCount"`

	// 抽烟报警拍驾驶员面部特征照片间隔时间
	// BYTE
	// 单位 100ms， 取值范围 1~5，默认 2
	// 0xFF 表示不修改参数
	SmokingAlarmDriverFacePhotoInterval uint8 `json:"smokingAlarmDriverFacePhotoInterval"`

	// 分神驾驶报警分级车速阈值
	// BYTE
	// 单位 km/h，取值范围 0~220，默认值 50。表示触发报
	// 警时车速高于阈值为二级报警，否则为一级报警
	// 0xFF 表示不修改参数
	DistractedDrivingSpeedThreshold uint8 `json:"distractedDrivingSpeedThreshold"`

	// 分神驾驶报警前后视频录制时间
	// BYTE
	// 单位秒，取值范围 0-60，默认值 5
	// 0 表示不录像，0xFF 表示不修改参数
	DistractedDrivingVideoRecordingTime uint8 `json:"distractedDrivingVideoRecordingTime"`

	// 分神驾驶报警拍照张数
	// BYTE
	// 取值范围 1-10，默认值 3
	// 0 表示不抓拍，0xFF 表示不修改参数
	DistractedDrivingPhotoCount uint8 `json:"distractedDrivingPhotoCount"`

	// 分神驾驶报警拍照间隔时间
	// BYTE
	// 单位 100ms， 取值范围 1~5，默认 2
	// 0xFF 表示不修改参数
	DistractedDrivingPhotoInterval uint8 `json:"distractedDrivingPhotoInterval"`

	// 驾驶行为异常分级速度阈值
	// BYTE
	// 单位 km/h，取值范围 0~220，默认值 50。表示触发报
	// 警时车速高于阈值为二级报警，否则为一级报警
	// 0xFF 表示不修改参数
	AbnormalDrivingSpeedThreshold uint8 `json:"abnormalDrivingSpeedThreshold"`

	// 驾驶行为异常视频录制时间
	// BYTE
	// 单位秒，取值范围 0-60，默认值 5
	// 0 表示不录像，0xFF 表示不修改参数
	AbnormalDrivingVideoRecordingTime uint8 `json:"abnormalDrivingVideoRecordingTime"`

	// 驾驶行为异常抓拍照片张数
	// BYTE
	// 取值范围 1-10，默认值 3
	// 0 表示不抓拍，0xFF 表示不修改参数
	AbnormalDrivingSnapPhotoCount uint8 `json:"abnormalDrivingSnapPhotoCount"`

	// 驾驶行为异常拍照间隔
	// BYTE
	// 单位 100ms， 取值范围 1~5，默认 2
	// 0xFF 表示不修改参数
	AbnormalDrivingSnapPhotoInterval uint8 `json:"abnormalDrivingSnapPhotoInterval"`

	// 驾驶员身份识别触发
	// BYTE
	// 0x00：不开启
	// 0x01：定时触发
	// 0x02：定距触发
	// 0x03：插卡开始行驶触发
	// 0x04：保留
	// 默认值为 0x01
	// 0xFF 表示不修改参数
	DriverIdentificationTrigger uint8 `json:"driverIdentificationTrigger"`

	// 保留字段
	// BYTE[2]
	ReservedField4 uint8 `json:"reservedField4"`
	ReservedField5 uint8 `json:"reservedField5"`
}
