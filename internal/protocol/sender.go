package protocol

import (
	"context"
	"errors"
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/fakeyanss/jt808-server-go/internal/storage"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"sync"
	"time"
)

type SenderKey struct {
	Phone        string `json:"phone"`
	MsgId        uint16 `json:"msgId"`
	SerialNumber uint16 `json:"serialNumber"`
}

type SenderValue struct {
	Phone            string          `json:"phone"`
	Msg              any             `json:"msg"`
	ResponseCallBack func(any) error `json:"-"`
}

// func (k *SenderKey) Equals(other *SenderKey) bool {
// 	return k.Phone == other.Phone &&
// 		k.MsgId == other.MsgId &&
// 		k.SerialNumber == other.SerialNumber
// }
//
// func (k *SenderKey) HashCode() uint32 {
// 	var sum uint32 = 0
// 	for i := 0; i < len(k.Phone); i++ {
// 		c := k.Phone[i]
// 		sum += uint32(c) * 31
// 	}
// 	sum += uint32(k.MsgId) * 31
// 	sum += uint32(k.SerialNumber) * 31
// 	return sum
// }

type Sender struct {
	Sending []*SenderValue             `json:"sending"` // 发送队列
	Waiting map[SenderKey]*SenderValue `json:"waiting"` // 等待相应队列
	Mutex   *sync.Mutex
}

func NewSender() *Sender {
	sender := &Sender{
		Sending: make([]*SenderValue, 0),
		Waiting: make(map[SenderKey]*SenderValue, 0),
		Mutex:   &sync.Mutex{},
	}
	return sender
}

func (s *Sender) Append(k *SenderKey, v *SenderValue) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if _, ok := s.Waiting[*k]; ok {
		err := errors.New("不允许重复发送消息")
		return err
	}

	s.Waiting[*k] = v
	s.Sending = append(s.Sending, v)
	return nil
}

// ProcResponse
// 由外部回調
// 处理由终端回复的响应数据
func (s *Sender) ProcResponse(phone string, ansMsgId uint16, ansSN uint16, msg any) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	key := &SenderKey{
		Phone:        phone,
		MsgId:        ansMsgId,
		SerialNumber: ansSN,
	}

	v, ok := s.Waiting[*key]
	if !ok {
		err := errors.New("没有找到对应的key")
		return err
	}

	delete(s.Waiting, *key)
	_ = v.ResponseCallBack(msg)
	return nil
}

// send
// 发送数据给终端
func (s *Sender) send() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	for i := 0; i < len(s.Sending); i++ {
		v := s.Sending[i]
		device, err := storage.GetDeviceCache().GetDeviceByPhone(v.Phone)
		if err != nil {
			continue
		}

		session, err := storage.GetSession(device.SessionID)
		if err != nil {
			continue
		}

		msg := v.Msg.(model.JT808Msg)
		pg := NewPipeline(session.Conn)

		// 记录value ctx
		ctx := context.WithValue(context.Background(), model.ProcessDataCtxKey{}, &model.ProcessData{Outgoing: msg})

		err = pg.ProcessConnWrite(ctx)
		if err != nil {
			if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed) {
				session.Conn.Close()
				storage.ClearSession(session.ID)
				// delete(serv.sessions, session.ID)

				log.Debug().Str("sessionId", session.ID).Msg("Closing connection from remote.")
			}

			log.Error().Err(err).Str("device", session.ID).Msg("Failed to send jtmsg to device")
			break
		}
	}

	s.Sending = s.Sending[:0]
	return nil
}

// 将sending队列中的数据发送出去
func (s *Sender) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			s.send()
			break
		}
	}
}
