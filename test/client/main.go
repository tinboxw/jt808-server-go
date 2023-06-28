package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/fakeyanss/jt808-server-go/internal/client"
	"github.com/fakeyanss/jt808-server-go/internal/config"
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/fakeyanss/jt808-server-go/internal/storage"
	"github.com/fakeyanss/jt808-server-go/pkg/logger"
	"github.com/fakeyanss/jt808-server-go/pkg/routines"
	"github.com/fakeyanss/jt808-server-go/test/datagen"
)

const (
	retryMaxCnt           = 600
	retryIntervalInSecond = 2
)

type (
	DeviceConfCtxKey struct{}

	DeviceGeoConfCtxKey struct{}

	DevicePhoneCtxKey struct{}
)

func buildDevice(ctx context.Context, cli *client.TCPClient) *model.Device {
	deviceConf := ctx.Value(DeviceConfCtxKey{}).(*config.DeviceConf)
	cache := storage.GetDeviceCache()
	device := datagen.GenDevice(deviceConf)
	device.SessionID = cli.Session.ID
	device.TransProto = model.TCPProto
	device.Conn = cli.Session.Conn
	cache.CacheDevice(device)
	return device
}

func getDevice(ctx context.Context) *model.Device {
	phone := ctx.Value(DevicePhoneCtxKey{}).(string)
	cache := storage.GetDeviceCache()
	device, err := cache.GetDeviceByPhone(phone)
	if err != nil {
		log.Fatal().Err(err).Str("phone", phone).Msg("Fail to find device cache")
	}
	return device
}

func register(ctx context.Context, cli *client.TCPClient) {
	device := getDevice(ctx)
	deviceConf := ctx.Value(DeviceConfCtxKey{}).(*config.DeviceConf)
	msg := datagen.GenMsg0100(deviceConf, device)
	cli.Send(msg)
}

func keepalive(ctx context.Context, cli *client.TCPClient) {
	device := getDevice(ctx)
	deviceConf := ctx.Value(DeviceConfCtxKey{}).(*config.DeviceConf)
	msg := datagen.GenMsg0002(device)

	for {
		cli.Send(msg)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(deviceConf.Keepalive) * time.Second):
		}
	}
}

func buildDeviceGeo(ctx context.Context) {
	device := getDevice(ctx)
	deviceGeoConf := ctx.Value(DeviceGeoConfCtxKey{}).(*config.DeviceGeoConf)
	deviceGeo := datagen.GenDeviceGeo(deviceGeoConf, device)
	geoCache := storage.GetGeoCache()
	rb := geoCache.GetGeoRingByPhone(device.Phone)
	rb.Write(deviceGeo)
}

func reportLocation(ctx context.Context, cli *client.TCPClient) {
	deviceGeoConf := ctx.Value(DeviceGeoConfCtxKey{}).(*config.DeviceGeoConf)
	geoCache := storage.GetGeoCache()

	for {
		device := getDevice(ctx)
		deviceGeo, err := geoCache.GetGeoLatestByPhone(device.Phone)
		if err == nil {
			msg := datagen.GenMsg0200(deviceGeoConf, device, deviceGeo)
			cli.Send(msg)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(deviceGeoConf.LocationReportInterval) * time.Second):
		}
	}
}

func dial(cli *client.TCPClient, addr string, pctx context.Context) {
	tctx, _ := context.WithCancel(pctx)
	for retry := 1; ; retry++ {
		err := cli.Dial(addr)
		if err == nil {
			break
		}
		errDialMsg := "Fail to dial to the tcp addr"
		if retry > retryMaxCnt {
			log.Error().Err(err).Str("addr", addr).Msgf("%s, exit", errDialMsg)
			os.Exit(1)
		}
		log.Error().Err(err).Str("addr", addr).Msgf("%s, retry", errDialMsg)

		select {
		case <-time.After(retryIntervalInSecond * time.Second):
			break
		case <-tctx.Done():
			goto done
		}
	}
done:
}

func do(cfg *config.Config,
	cliWg *sync.WaitGroup,
	cli *client.TCPClient,
	pctx context.Context) {

	tctx, cancel := context.WithCancel(pctx)
	routines.GoSafe(func() {
		defer cliWg.Done()

		log.Debug().Msgf("start tcp client...")
		cli.Start()

		cancel()
	})

	routines.GoSafe(func() {
		ctx := context.WithValue(tctx, DeviceConfCtxKey{}, cfg.Client.Device)
		d := buildDevice(ctx, cli)
		ctx = context.WithValue(ctx, DevicePhoneCtxKey{}, d.Phone)
		ctx = context.WithValue(ctx, DeviceGeoConfCtxKey{}, cfg.Client.DeviceGeo)
		buildDeviceGeo(ctx)

		var wg sync.WaitGroup
		wg.Add(1)

		routines.GoSafe(func() {
			// 首次注册
			register(ctx, cli)
			log.Debug().Msgf("sent register msg done")

			// device status checker
			for {
				select {
				case <-time.After(10 * time.Second):
					// 10秒没有上线，就重新注册
					register(ctx, cli)
					log.Debug().Msgf("sent register msg done")
					break
				case <-time.After(time.Second):
					// 每秒检查一次
					cache := storage.GetDeviceCache()
					renewDevice, _ := cache.GetDeviceByPhone(d.Phone)
					if renewDevice.Status == model.DeviceStatusOnline {
						wg.Done()
						log.Debug().Msgf("client is online")
						return
					}
					log.Debug().Msgf("waiting for client online, sleep...")
					break
				}
			}
		})

		// should wait for register success, and stop after register failed for a while
		routines.GoSafe(func() {
			wg.Wait()
			log.Debug().Msgf("start keepalive loop...")
			keepalive(ctx, cli)
		})
		routines.GoSafe(func() {
			wg.Wait()
			log.Debug().Msgf("start report location msg loop...")
			reportLocation(ctx, cli)
		})
	})
}

func dialAndSend(cfg *config.Config, cliWg *sync.WaitGroup, tctx context.Context) {
	for i := 0; i < cfg.Client.Concurrency; i++ {
		cli := client.NewTCPClient()
		addr := cfg.Client.Conn.RemoteAddr

		dial(cli, addr, tctx)
		do(cfg, cliWg, cli, tctx)
	}
}

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "c", config.DefaultCliConfKey, "config file path")
	flag.Parse()
	fmt.Printf("Start with configuration %v\n", cfgPath)
	cfg := config.Load(cfgPath)

	// 设置工作路径
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	_ = os.Chdir(dir + "/../")

	remote := os.Getenv("REMOTE")
	if len(remote) != 0 {
		cfg.Client.Conn.RemoteAddr = remote
	}

	logCfg := config.ParseLoggerConfig(cfg.Log)
	log.Logger = *logger.Configure(logCfg).Logger

	var cliWg sync.WaitGroup
	cliWg.Add(cfg.Client.Concurrency)
	tctx, _ := context.WithCancel(context.Background())
	dialAndSend(cfg, &cliWg, tctx)
	cliWg.Wait()

	for {
		select {
		case <-time.After(retryIntervalInSecond * time.Second):
			cliWg.Add(cfg.Client.Concurrency)
			dialAndSend(cfg, &cliWg, tctx)
			cliWg.Wait()
			break
		case <-tctx.Done():
			return
		}
	}
}
