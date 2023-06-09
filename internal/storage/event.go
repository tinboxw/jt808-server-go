package storage

import "github.com/fakeyanss/jt808-server-go/internal/protocol/model"

type StatusChangeHandler func(model.DeviceStatus) error

type ReportAlarmMsgHandler func(msg *model.AlarmMsg) error
