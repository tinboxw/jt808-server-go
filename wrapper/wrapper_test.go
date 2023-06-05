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
