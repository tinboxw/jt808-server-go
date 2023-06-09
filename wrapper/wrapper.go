package wrapper

import (
	"encoding/json"
	"github.com/fakeyanss/jt808-server-go/internal/config"
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/fakeyanss/jt808-server-go/internal/server"
	"github.com/fakeyanss/jt808-server-go/internal/storage"
	"github.com/fakeyanss/jt808-server-go/pkg/logger"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"time"
)

type LogLevelType = config.LogLevelType
type LogConf = config.LogConf

type Jt808Server struct {
	server.Server
}

type result struct {
	phone string
	code  uint8
	desc  string
}

func New() *Jt808Server {
	srv := &Jt808Server{Server: server.NewTCPServer()}
	return srv
}

type EventType = int

const (
	EventInit         EventType = iota
	EventStatusChange           // offline/online/sleeping
	EventReportAlarm
	EventReportEvent

	EventMax
)

type EventHandler func(any) error
type DeviceStatus = model.DeviceStatus

// type AlarmMsg = model.AlarmMsg

const (
	DeviceStatusOffline DeviceStatus = iota
	DeviceStatusOnline
	DeviceStatusSleeping
)

type EventHandlers struct {
	OnDeviceStatusChange func(DeviceStatus) error
	// json string
	OnReportAlarmMsg func([]byte) error
}

func (s *Jt808Server) SetLogger(c *LogConf) {
	logConfig := config.ParseLoggerConfig(c)
	log.Logger = *logger.Configure(logConfig).Logger
}

func (s *Jt808Server) SetHooks(handlers EventHandlers) error {
	device := storage.GetDeviceCache()
	device.SetStatusHook(func(status model.DeviceStatus) error {
		handlers.OnDeviceStatusChange(status)
		return nil
	})
	cache := storage.GetDeviceAlarmMsgCache()
	cache.SetAlarmMsgHook(func(msg *model.AlarmMsg) error {
		if msg == nil {
			return nil
		}

		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		return handlers.OnReportAlarmMsg(data)
	})

	return nil
}

// GetDeviceConfig
// 获取设备参数配置，如果存在多个设备，只会将最后一个返回
func (s *Jt808Server) GetDeviceConfig(to int) ([]byte, error) {
	var settings []byte
	var err error

	err = s.send2Devices(0x8104, func(header *model.MsgHeader) model.JT808Msg {
		msg := &model.Msg8104{
			Header: header,
		}
		return msg
	}, func(m any) error {
		rsp := m.(*model.Msg0104)
		r := result{
			phone: rsp.GetHeader().PhoneNumber,
		}

		// TODO::始终获取最后一份数据
		// 因为我自己使用的场景，只会有一台终端
		settings, err = json.Marshal(rsp.Parameters)
		if err != nil {
			r.code = 1
			r.desc = err.Error()
		}

		log.Debug().
			Msgf("执行0x8104消息返回结果：%s => %d,%s", r.phone, r.code, r.desc)

		return err
	}, to)

	return settings, err
}

// SetDeviceConfig
// json string
func (s *Jt808Server) SetDeviceConfig(data []byte, to int) error {
	var params = &model.DeviceParams{}
	err := json.Unmarshal(data, params)
	if err != nil {
		// TODO:: logger
		return err
	}

	err = s.send2Devices(0x8103, func(header *model.MsgHeader) model.JT808Msg {
		msg := &model.Msg8103{
			Header:     header,
			Parameters: params,
		}
		return msg
	}, func(m any) error {
		rsp := m.(*model.Msg0001)
		r := result{
			phone: rsp.GetHeader().PhoneNumber,
			code:  rsp.Result,
			desc:  model.Msg0001Result2Str(rsp.Result),
		}
		log.Debug().
			Msgf("执行0x8103消息返回结果：%s => %d,%s", r.phone, r.code, r.desc)
		if r.code != 0 {
			err := errors.Wrapf(nil, "%s:%d,%s", r.phone, r.code, r.desc)
			return err
		}
		return nil
	}, to)

	return err
}

func (s *Jt808Server) send2Devices(msgId uint16,
	buildMsgFn func(msg *model.MsgHeader) model.JT808Msg,
	procRspFn func(m any) error,
	to int) error {
	cache := storage.GetDeviceCache()

	// 我们现在的使用情况来看，只会有一个设备
	all := cache.ListDevice()
	total := len(all)
	if total == 0 {
		err := errors.New("no devices, set device config failed")
		return err
	}

	waitChan := make(chan error, total)
	for _, device := range all {
		session, err := storage.GetSession(device.SessionID)
		if err != nil {
			// TODO:: logger
			continue
		}

		header := model.GenMsgHeader(device, msgId, session.GetNextSerialNum())
		msg := buildMsgFn(header)

		_ = s.SendV2(device.Phone, msg, func(m any) error {
			err := procRspFn(m)
			waitChan <- err
			return nil
		})
	}

	cnt := 0
	var err error = nil
	for {
		select {
		case <-time.After(time.Second * time.Duration(to)):
			err = errors.Errorf("超时，没有接收完所有设置的响应：recv:%d, expect:%d", cnt, total)
			log.Error().
				Msgf(err.Error())
			goto END
			break
		case r := <-waitChan:
			if r != nil {
				if err == nil {
					err = r
				} else {
					err = errors.Wrapf(err, "%v", r)
				}
			}
			cnt++
			if cnt >= total {
				log.Debug().
					Msgf("已经接收完所有结果：recv:%d, expect:%d", cnt, total)
				goto END
			}
			break
		}
	}
END:
	return err
}

// GetEnumName 获取枚举值对应的枚举字符名
func GetEnumName(v interface{}) string {
	switch v {
	case model.DeviceStatusOffline:
		return "offline"
	case model.DeviceStatusOnline:
		return "online"
	case model.DeviceStatusSleeping:
		return "sleeping"
	default:
		return "unknown"
	}

	// enumType := reflect.TypeOf(enumValue)
	// enumValueInt := int(reflect.ValueOf(enumValue).Int())
	//
	// for i := 0; i < enumType.NumField(); i++ {
	//	enumField := enumType.Field(i)
	//	if enumField.Type.Kind() == reflect.Int && enumField.Tag.Get("enum") != "" {
	//		tagValue := enumField.Tag.Get("enumValueInt")
	//		valueInt, err := strconv.Atoi(tagValue)
	//		if err == nil && valueInt == enumValueInt {
	//			return enumField.Tag.Get("enum")
	//		}
	//	}
	// }
	//
	// return "Unknown"
}
