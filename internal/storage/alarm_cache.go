package storage

import (
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/pkg/errors"
	"sync"
)

var ErrDeviceAlarmMsgNotFound = errors.New("device alarm msg not found")

type DeviceAlarmMsgCache struct {
	// caches
	// 保存的是设备上报的最后一条告警记录
	cacheByPhone map[string]any

	// 当产生告警时的回调函数
	hook ReportAlarmMsgHandler

	// 锁
	mutex *sync.Mutex
}

var alarmMsgCacheSingleton *DeviceAlarmMsgCache
var alarmMsgCacheInitOnce sync.Once

func GetDeviceAlarmMsgCache() *DeviceAlarmMsgCache {
	alarmMsgCacheInitOnce.Do(func() {
		alarmMsgCacheSingleton = &DeviceAlarmMsgCache{
			cacheByPhone: make(map[string]any),
			mutex:        &sync.Mutex{},
		}
	})
	return alarmMsgCacheSingleton
}

func (cache *DeviceAlarmMsgCache) SetAlarmMsgHook(handler ReportAlarmMsgHandler) {
	cache.hook = handler
}

func (cache *DeviceAlarmMsgCache) GetDeviceAlarmMsgByPhone(phone string) (any, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if d, ok := cache.cacheByPhone[phone]; ok {
		return d, nil
	}
	return nil, ErrDeviceAlarmMsgNotFound
}

func (cache *DeviceAlarmMsgCache) CacheDeviceAlarmMsg(phone string, msg *model.AlarmMsg) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// 回调
	if cache.hook != nil {
		_ = cache.hook(msg)
	}

	// 保存
	cache.cacheByPhone[phone] = msg
}

func (cache *DeviceAlarmMsgCache) DelDeviceAlarmMsgByPhone(phone string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	_, ok := cache.cacheByPhone[phone]
	if !ok {
		return // find none device params, skip
	}
	delete(cache.cacheByPhone, phone)
}
