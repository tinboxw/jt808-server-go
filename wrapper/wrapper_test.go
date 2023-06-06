package wrapper

import (
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

	s.SetHooks(EventStatusChange, func(s any) error {
		status := s.(int)
		t.Logf("the device status change to:%d,%s", status, GetEnumName(status))
		return nil
	})

	s.SetHooks(EventReportAlarm, func(s any) error {
		msg := s.([]byte)
		assert.NotNil(t, msg)

		t.Logf("recv alarm msg:%s", msg)
		return nil
	})

	err := s.Listen("localhost:8080")
	assert.Nil(t, err)

	go func() {
		s.Start()
	}()

	time.Sleep(100 * time.Second)
	s.Stop()
}
