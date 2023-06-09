package model

import (
	"github.com/fakeyanss/jt808-server-go/internal/codec/hex"
)

// 终端通用应答
type Msg0001 struct {
	Header             *MsgHeader `json:"header"`
	AnswerSerialNumber uint16     `json:"answerSerialNumber"` // 2位，应答流水号，对应平台消息的流水号，
	AnswerMessageID    uint16     `json:"answerMessageId"`    // 2位，应答ID，对应平台消息的ID
	Result             uint8      `json:"result"`             // 1位，结果，0成功/确认，1失败，2消息有误，3不支持
}

func Msg0001Result2Str(res uint8) string {
	// 1位，结果，0成功/确认，1失败，2消息有误，3不支持
	switch res {
	case 0:
		return "success"
	case 1:
		return "failed"
	case 2:
		return "invalid parameters"
	case 3:
		return "unsupported"
	}

	return "unknown"
}

func (m *Msg0001) Result2Str() string {
	return Msg0001Result2Str(m.Result)
}

func (m *Msg0001) Decode(packet *PacketData) error {
	m.Header = packet.Header
	pkt, idx := packet.Body, 0

	m.AnswerSerialNumber = hex.ReadWord(pkt, &idx)
	m.AnswerMessageID = hex.ReadWord(pkt, &idx)
	m.Result = hex.ReadByte(pkt, &idx)
	return nil
}

func (m *Msg0001) Encode() (pkt []byte, err error) {
	pkt = hex.WriteWord(pkt, m.AnswerSerialNumber)
	pkt = hex.WriteWord(pkt, m.AnswerMessageID)
	pkt = hex.WriteByte(pkt, m.Result)

	pkt, err = writeHeader(m, pkt)
	return pkt, err
}

func (m *Msg0001) GetHeader() *MsgHeader {
	return m.Header
}

func (m *Msg0001) GenOutgoing(incoming JT808Msg) error {
	header := incoming.GetHeader()
	m.AnswerSerialNumber = header.SerialNumber
	m.AnswerMessageID = header.MsgID
	m.Result = 0

	m.Header = header
	m.Header.MsgID = 0x0001
	return nil
}
