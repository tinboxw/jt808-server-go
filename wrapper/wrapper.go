package wrapper

import (
	"encoding/json"
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/fakeyanss/jt808-server-go/internal/server"
	"github.com/fakeyanss/jt808-server-go/internal/storage"
	"github.com/pkg/errors"
)

type Jt808Server struct {
	server.Server
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

func (s *Jt808Server) SetHooks(ev EventType, handler EventHandler) error {
	switch ev {
	case EventStatusChange:
		device := storage.GetDeviceCache()
		device.SetStatusHook(func(status any) error {
			s := status.(model.DeviceStatus)
			switch s {
			case model.DeviceStatusOffline:
				return handler(0)
			case model.DeviceStatusOnline:
				return handler(1)
			case model.DeviceStatusSleeping:
				return handler(3)
			}
			return nil
		})
		break
	case EventReportAlarm:
		cache := storage.GetDeviceAlarmMsgCache()
		cache.SetAlarmMsgHook(func(msg any) error {
			data, err := json.Marshal(msg)
			if err != nil {
				return err
			}

			return handler(data)
		})
	default:
		err := errors.New("unsupported event handler")
		return err
	}

	return nil
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

	//enumType := reflect.TypeOf(enumValue)
	//enumValueInt := int(reflect.ValueOf(enumValue).Int())
	//
	//for i := 0; i < enumType.NumField(); i++ {
	//	enumField := enumType.Field(i)
	//	if enumField.Type.Kind() == reflect.Int && enumField.Tag.Get("enum") != "" {
	//		tagValue := enumField.Tag.Get("enumValueInt")
	//		valueInt, err := strconv.Atoi(tagValue)
	//		if err == nil && valueInt == enumValueInt {
	//			return enumField.Tag.Get("enum")
	//		}
	//	}
	//}
	//
	//return "Unknown"
}
