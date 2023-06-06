package config

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/mix-go/xfmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/fakeyanss/jt808-server-go/pkg/logger"
)

type LogLevelType string

const (
	LogLevelFatal LogLevelType = "FATAl"
	LogLevelError LogLevelType = "ERROR"
	LogLevelWarn  LogLevelType = "WARN"
	LogLevelInfo  LogLevelType = "INFO"
	LogLevelDebug LogLevelType = "DEBUG"
)

const (
	DefaultServConfKey  string = "embedded-default-server-config"
	DefaultServConfPath string = "configs/default.yaml"
	DefaultCliConfKey   string = "embedded-default-client-config"
	DefaultCliConfPath  string = "test/client/configs/default.yaml"
)

type LogConf struct {
	ConsoleEnable       bool         `yaml:"consoleEnable"  json:"consoleEnable"`
	FileEnable          bool         `yaml:"fileEnable"  json:"fileEnable"`
	PrintAsJSON         bool         `yaml:"printAsJson"  json:"printAsJson"`
	LogLevel            LogLevelType `yaml:"logLevel"  json:"logLevel"`
	LogDirectory        string       `yaml:"logDirectory"  json:"logDirectory"`
	LogFile             string       `yaml:"logFile"  json:"logFile"`
	MaxSizeOfRolling    int          `yaml:"maxSizeOfRolling"  json:"maxSizeOfRolling"`
	MaxBackupsOfRolling int          `yaml:"maxBackupsOfRolling"  json:"maxBackupsOfRolling"`
	MaxAgeOfRolling     int          `yaml:"maxAgeOfRolling"  json:"maxAgeOfRolling"`
}

type serverConf struct {
	Name   string      `yaml:"name" json:"name"`
	Port   *servPort   `yaml:"port" json:"port"`
	Banner *servBanner `yaml:"banner" json:"banner"`
}

type servPort struct {
	TCPPort  string `yaml:"tcpPort" json:"tcpPort"`
	UDPPort  string `yaml:"udpPort" json:"udpPort"`
	HTTPPort string `yaml:"httpPort" json:"httpPort"`
}

type servBanner struct {
	Enable     bool   `yaml:"enable" json:"enable"`
	BannerPath string `yaml:"bannerPath" json:"bannerPath"`
}

type clientConf struct {
	Name         string            `yaml:"name" json:"name"`
	Conn         *connection       `yaml:"conn" json:"conn"`
	Concurrency  int               `yaml:"concurrency" json:"concurrency"`
	Device       *DeviceConf       `yaml:"device" json:"device"`
	DeviceGeo    *DeviceGeoConf    `yaml:"deviceGeo" json:"deviceGeo"`
	DeviceParams *DeviceParamsConf `yaml:"deviceParams" json:"deviceParams"`
}

type connection struct {
	RemoteAddr string `yaml:"remoteAddr" json:"remoteAddr"`
}

type DeviceConf struct {
	IDReg           string `yaml:"idReg" json:"idReg,omitempty"`
	IMEIReg         string `yaml:"imeiReg" json:"imeiReg,omitempty"`
	PhoneReg        string `yaml:"phoneReg" json:"phoneReg,omitempty"`
	PlateReg        string `yaml:"plateReg" json:"plateReg,omitempty"`
	ProtocolVersion string `yaml:"protocolVersion" json:"protocolVersion,omitempty"`
	TransProto      string `yaml:"transProto" json:"transProto,omitempty"`
	Keepalive       int    `yaml:"keepalive" json:"keepalive,omitempty"`
	ProvinceIDReg   string `yaml:"provinceIdReg" json:"provinceIDReg,omitempty"`
	CityIDReg       string `yaml:"cityIdReg" json:"cityIDReg,omitempty"`
	PlateColorReg   string `yaml:"plateColorReg" json:"plateColorReg,omitempty"`
}

type DeviceGeoConf struct {
	LocationReportInterval int           `yaml:"locationReportInterval" json:"locationReportInterval,omitempty"`
	Geo                    *geoConf      `yaml:"geo" json:"geo,omitempty"`
	Location               *locationConf `yaml:"location" json:"location,omitempty"`
	Drive                  *driveConf    `yaml:"drive" json:"drive,omitempty"`
	Expand                 []expandConf  `yaml:"expand,flow" json:"expand,omitempty"`
}

type expandConf struct {
	Id uint8 `yaml:"id" json:"id,omitempty"`
	//Length  uint8  `yaml:"length"`// 长度通过 payload 重新计算
	Value string `yaml:"value" json:"value,omitempty"` // hex数组，注意：是hex数组字符串，每两个字符为一个字节
}

type geoConf struct {
	ACCStatusReg              string `yaml:"accStatusReg" json:"accStatusReg,omitempty"`
	LocationStatusReg         string `yaml:"locationStatusReg" json:"locationStatusReg,omitempty"`
	LatitudeTypeReg           string `yaml:"latitudeTypeReg" json:"latitudeTypeReg,omitempty"`
	LongitudeTypeReg          string `yaml:"longitudeTypeReg" json:"longitudeTypeReg,omitempty"`
	OperatingStatusReg        string `yaml:"operatingStatusReg" json:"operatingStatusReg,omitempty"`
	GeoEncryptionStatusReg    string `yaml:"geoEncryptionStatusReg" json:"geoEncryptionStatusReg,omitempty"`
	LoadStatusReg             string `yaml:"loadStatusReg" json:"loadStatusReg,omitempty"`
	FuelSystemStatusReg       string `yaml:"fuelSystemStatusReg" json:"fuelSystemStatusReg,omitempty"`
	AlternatorSystemStatusReg string `yaml:"alternatorSystemStatusReg" json:"alternatorSystemStatusReg,omitempty"`
	DoorLockedStatusReg       string `yaml:"doorLockedStatusReg" json:"doorLockedStatusReg,omitempty"`
	FrontDoorStatusReg        string `yaml:"frontDoorStatusReg" json:"frontDoorStatusReg,omitempty"`
	MidDoorStatusReg          string `yaml:"midDoorStatusReg" json:"midDoorStatusReg,omitempty"`
	BackDoorStatusReg         string `yaml:"backDoorStatusReg" json:"backDoorStatusReg,omitempty"`
	DriverDoorStatusReg       string `yaml:"driverDoorStatusReg" json:"driverDoorStatusReg,omitempty"`
	CustomDoorStatusReg       string `yaml:"customDoorStatusReg" json:"customDoorStatusReg,omitempty"`
	GPSLocationStatusReg      string `yaml:"gpsLocationStatusReg" json:"GPSLocationStatusReg,omitempty"`
	BeidouLocationStatusReg   string `yaml:"beidouLocationStatusReg" json:"beidouLocationStatusReg,omitempty"`
	GLONASSLocationStatusReg  string `yaml:"glonassLocationStatusReg" json:"GLONASSLocationStatusReg,omitempty"`
	GalileoLocationStatusReg  string `yaml:"galileoLocationStatusReg" json:"galileoLocationStatusReg,omitempty"`
	DrivingStatusReg          string `yaml:"drivingStatusReg" json:"drivingStatusReg,omitempty"`
}

type locationConf struct {
	LatitudeReg  string `yaml:"latitudeReg" json:"latitudeReg,omitempty"`
	LongitudeReg string `yaml:"longitudeReg" json:"longitudeReg,omitempty"`
	AltitudeReg  string `yaml:"altitudeReg" json:"altitudeReg,omitempty"`
}

type driveConf struct {
	SpeedReg     string `yaml:"speedReg" json:"speedReg,omitempty"`
	DirectionReg string `yaml:"directionReg" json:"directionReg,omitempty"`
}

type DeviceParamsConf struct {
}

type Config struct {
	Log    *LogConf    `yaml:"log" json:"log"`
	Server *serverConf `yaml:"server" json:"server"`
	Client *clientConf `yaml:"client" json:"client"`
}

var (
	configOnce sync.Once
	config     *Config
)

func Load(confFilePath string) *Config {
	configOnce.Do(func() {
		config = &Config{}
		viper.SetConfigType("yaml")

		var err error
		if confFilePath == DefaultServConfKey || confFilePath == DefaultCliConfKey {
			// replace default embedded conf path
			if confFilePath == DefaultServConfKey {
				confFilePath = DefaultServConfPath
			} else if confFilePath == DefaultCliConfKey {
				confFilePath = DefaultCliConfPath
			}
			var confContent []byte
			confContent, err = Asset(confFilePath)
			if err != nil {
				panic(errors.Wrap(err, "Fail to read default config with bindata"))
			}
			err = viper.ReadConfig(bytes.NewBuffer(confContent))
		} else {
			viper.SetConfigFile(confFilePath)
			err = viper.ReadInConfig()
		}

		if err != nil {
			panic(errors.Wrap(err, "Fail to read config with viper"))
		}

		err = viper.Unmarshal(config)
		if err != nil {
			panic(errors.Wrap(err, "Fail to unmarshal config"))
		}
		fmt.Printf("Load configuration: %s\n", xfmt.Sprintf("%+v", config))
	})
	return config
}

func ParseLoggerConfig(logCfg *LogConf) *logger.Config {
	var logLevel int8
	switch logCfg.LogLevel {
	case "DEBUG":
		logLevel = int8(zerolog.DebugLevel)
	case "INFO":
		logLevel = int8(zerolog.InfoLevel)
	case "WARN":
		logLevel = int8(zerolog.WarnLevel)
	case "ERROR":
		logLevel = int8(zerolog.ErrorLevel)
	case "FATAl":
		logLevel = int8(zerolog.FatalLevel)
	}
	return &logger.Config{
		ConsoleLoggingEnabled: logCfg.ConsoleEnable,
		EncodeLogsAsJSON:      logCfg.PrintAsJSON,
		FileLoggingEnabled:    logCfg.FileEnable,
		LogLevel:              logLevel,
		Directory:             logCfg.LogDirectory,
		Filename:              logCfg.LogFile,
		MaxSize:               logCfg.MaxSizeOfRolling,
		MaxBackups:            logCfg.MaxBackupsOfRolling,
		MaxAge:                logCfg.MaxAgeOfRolling,
	}
}
