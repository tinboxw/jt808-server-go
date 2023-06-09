package storage

import (
	"errors"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
)

var ErrDeviceNotFound = errors.New("device not found")

type DeviceCache struct {
	cacheByPlate map[string]*model.Device
	cacheByPhone map[string]*model.Device

	// 当产生告警时的回调函数
	hook StatusChangeHandler

	// 锁
	mutex *sync.Mutex
}

var deviceCacheSingleton *DeviceCache
var deviceCacheInitOnce sync.Once

func GetDeviceCache() *DeviceCache {
	deviceCacheInitOnce.Do(func() {
		deviceCacheSingleton = &DeviceCache{
			cacheByPlate: make(map[string]*model.Device),
			cacheByPhone: make(map[string]*model.Device),
			mutex:        &sync.Mutex{},
		}
	})
	return deviceCacheSingleton
}

func (cache *DeviceCache) ListDevice() []*model.Device {
	return maps.Values(cache.cacheByPhone)
}

func (cache *DeviceCache) GetDeviceByPlate(carPlate string) (*model.Device, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if d, ok := cache.cacheByPlate[carPlate]; ok {
		return d, nil
	}
	return nil, ErrDeviceNotFound
}

func (cache *DeviceCache) HasPlate(carPlate string) bool {
	d, err := cache.GetDeviceByPlate(carPlate)
	return d != nil && err == nil
}

func (cache *DeviceCache) GetDeviceByPhone(phone string) (*model.Device, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if d, ok := cache.cacheByPhone[phone]; ok {
		return d, nil
	}
	return nil, ErrDeviceNotFound
}

func (cache *DeviceCache) HasPhone(phone string) bool {
	d, err := cache.GetDeviceByPhone(phone)
	return d != nil && err == nil
}

func (cache *DeviceCache) cacheDevice(d *model.Device) {
	cache.cacheByPlate[d.Plate] = d
	cache.cacheByPhone[d.Phone] = d
}

func (cache *DeviceCache) CacheDevice(d *model.Device) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.cacheDevice(d)
}

func (cache *DeviceCache) SetStatusHook(handler StatusChangeHandler) {
	cache.hook = handler
}

func (cache *DeviceCache) UpdateDeviceStatus(d *model.Device, to model.DeviceStatus) {
	if cache.hook != nil {
		_ = cache.hook(to)
	}

	d.Status = to
	cache.CacheDevice(d)
}

func (cache *DeviceCache) delDevice(carPlate, phone *string) {
	var d *model.Device
	var ok bool
	if carPlate != nil {
		d, ok = cache.cacheByPlate[*carPlate]
	}
	if phone != nil {
		d, ok = cache.cacheByPhone[*phone]
	}
	if !ok {
		return // find none device, skip
	}
	delete(cache.cacheByPlate, d.Plate)
	delete(cache.cacheByPhone, d.Phone)
}

func (cache *DeviceCache) DelDeviceByCarPlate(carPlate string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.delDevice(&carPlate, nil)
}

func (cache *DeviceCache) DelDeviceByPhone(phone string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.delDevice(nil, &phone)
}
