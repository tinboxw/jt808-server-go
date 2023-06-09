package wrapper

import (
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New()
	assert.NotNil(t, s)

	err := s.Listen("localhost:8088")
	assert.Nil(t, err)

	go func() {
		s.Start()
	}()

	time.Sleep(10 * time.Second)
	s.Stop()
}

func TestJt808Server_SetHooks(t *testing.T) {
	s := New()
	assert.NotNil(t, s)

	handlers := EventHandlers{
		OnDeviceStatusChange: func(status model.DeviceStatus) error {
			// status := s.(int)
			t.Logf("the device status change to:%d,%s", status, GetEnumName(status))
			return nil
		},
		OnReportAlarmMsg: func(data []byte) error {
			// msg := s.([]byte)
			assert.NotNil(t, data)

			t.Logf("recv alarm msg:%s", data)
			return nil
		},
	}
	s.SetHooks(handlers)

	err := s.Listen("localhost:8080")
	assert.Nil(t, err)

	go func() {
		s.Start()
	}()

	time.Sleep(100 * time.Second)
	s.Stop()
}

func TestJt808Server_SetDeviceConfig(t *testing.T) {
	s := New()
	assert.NotNil(t, s)

	err := s.Listen("localhost:8080")
	assert.Nil(t, err)

	waitOnline := make(chan model.DeviceStatus)
	handlers := EventHandlers{
		OnDeviceStatusChange: func(status model.DeviceStatus) error {
			// status := s.(model.DeviceStatus)
			waitOnline <- status
			t.Logf("the device status change to:%d,%s", status, GetEnumName(status))
			return nil
		},
		OnReportAlarmMsg: func(data []byte) error {
			// msg := s.([]byte)
			assert.NotNil(t, data)

			t.Logf("recv alarm msg:%s", data)
			return nil
		},
	}
	s.SetHooks(handlers)

	go func() {
		s.Start()
	}()

	for {
		select {
		case status := <-waitOnline:
			if status == model.DeviceStatusOnline {
				t.Logf("the device is online")
				goto NEXT
			}
			break
		case <-time.After(time.Second * 100):
			t.Error("no device online")
			goto NEXT
		}
	}

NEXT:
	conf := "{\"settings\":[{\"id\":1,\"value\":10,\"desc\":\"0x0001, 终端心跳发送间隔，单位为秒（s）\"},{\"id\":19,\"value\":\"192.168.3.159:8080\",\"desc\":\"0x0013, jt808-2013:主服务器地址,IP或域名; jt808-2019:主服务器地址,IP或域名,以冒号分割主机和端口,多个服务器使用分号分割\"},{\"id\":23,\"value\":\"192.168.3.154:8080\",\"desc\":\"0x0017, jt808-2013:主服务器地址,IP或域名; jt808-2019:主服务器地址,IP或域名,以冒号分割主机和端口,多个服务器使用分号分割\"},{\"id\":24,\"value\":8080,\"desc\":\"0x0018, jt808-2013:服务器tcp端口；jt808-2019：不开放该字段，使用0x0013/0x0017中端口\"},{\"id\":25,\"value\":8080,\"desc\":\"0x0019, jt808-2013:服务器tcp端口；jt808-2019：不开放该字段，使用0x0013/0x0017中端口\"},{\"id\":62309,\"desc\":\"0xF365, DSM设置,不进行修改时字段都应该赋值为0xFF/0xFFFF/0xFFFFFFFF\",\"value\":{\"alarmSpeedThreshold\":255,\"alarmVolume\":255,\"proactivePhotoStrategy\":255,\"proactivePhotoInterval\":65535,\"proactivePhotoDistanceInterval\":65535,\"proactivePhotoCount\":255,\"proactivePhotoIntervalTime\":255,\"photoResolution\":255,\"videoResolution\":255,\"alarmEnabled\":4294967295,\"eventEnabled\":4294967295,\"smokingAlarmInterval\":65535,\"phoneCallAlarmInterval\":65535,\"reservedField1\":0,\"reservedField2\":0,\"reservedField3\":0,\"fatigueDrivingSpeedThreshold\":255,\"fatigueDrivingVideoRecordingTime\":255,\"fatigueDrivingPhotoCount\":255,\"fatigueDrivingPhotoInterval\":255,\"phoneCallAlarmSpeedThreshold\":255,\"phoneCallAlarmVideoRecordingTime\":255,\"phoneCallDriverFacePhotoCount\":255,\"phoneCallDriverFaceFeatureInterval\":255,\"smokingAlarmSpeedThreshold\":255,\"smokingAlarmVideoRecordingTime\":255,\"smokingAlarmDriverFacePhotoCount\":255,\"smokingAlarmDriverFacePhotoInterval\":255,\"distractedDrivingSpeedThreshold\":255,\"distractedDrivingVideoRecordingTime\":255,\"distractedDrivingPhotoCount\":255,\"distractedDrivingPhotoInterval\":255,\"abnormalDrivingSpeedThreshold\":255,\"abnormalDrivingVideoRecordingTime\":255,\"abnormalDrivingSnapPhotoCount\":255,\"abnormalDrivingSnapPhotoInterval\":255,\"driverIdentificationTrigger\":255,\"reservedField4\":0,\"reservedField5\":0}}]}"
	err = s.SetDeviceConfig([]byte(conf), 30)
	assert.Nil(t, err)

	// time.Sleep(100 * time.Second)
	s.Stop()
}

func TestJt808Server_GetDeviceConfig(t *testing.T) {
	s := New()
	assert.NotNil(t, s)

	err := s.Listen("localhost:8080")
	assert.Nil(t, err)

	waitOnline := make(chan model.DeviceStatus)
	handlers := EventHandlers{
		OnDeviceStatusChange: func(status model.DeviceStatus) error {
			// status := s.(model.DeviceStatus)
			waitOnline <- status
			t.Logf("the device status change to:%d,%s", status, GetEnumName(status))
			return nil
		},
		OnReportAlarmMsg: func(data []byte) error {
			// msg := s.([]byte)
			assert.NotNil(t, data)

			t.Logf("recv alarm msg:%s", data)
			return nil
		},
	}
	s.SetHooks(handlers)

	go func() {
		s.Start()
	}()

	for {
		select {
		case status := <-waitOnline:
			if status == model.DeviceStatusOnline {
				t.Logf("the device is online")
				goto NEXT
			}
			break
		case <-time.After(time.Second * 100):
			t.Error("no device online")
			goto NEXT
		}
	}

NEXT:
	conf, err := s.GetDeviceConfig(30)
	assert.Nil(t, err)
	t.Logf("get config:%s", conf)

	// time.Sleep(100 * time.Second)
	s.Stop()
}
