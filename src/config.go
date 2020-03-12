package main

import (
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

const (
	defaultAddr       = ":8888"
	defaultConfigFile = "config.yml"

	// Time allowed to write a message to the peer.
	defaultWriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	defaultPongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	defaultPingPeriod = (defaultPongWait * 9) / 10

	// Maximum message size allowed from peer.
	defaultMaxMessageSize = 1024 * 4

	defaultReadBufferSize  = 1024
	defaultWriteBufferSize = 1024
)

type ConfigFile struct {
	Server struct {
		Addr string `yaml:"addr", envconfig:"SERVER_ADDR"`
	} `yaml:"server"`

	Session struct {
		WriteWait      string `yaml:"writeWait"`
		PongWait       string `yaml:"pongWait"`
		MaxMessageSize int64  `yaml:"maxMessageSize"`
	} `yaml:"session"`

	Upgrader struct {
		ReadBufferSize  int64 `yaml:"readBufferSize"`
		WriteBufferSize int64 `yaml:"writeBufferSize"`
	} `yaml:"upgrader"`
}

type SessionConfig struct {
	writeWait  time.Duration
	pongWait   time.Duration
	pingPeriod time.Duration

	maxMessageSize int64
}

type UpgraderConfig struct {
	readBufferSize  int
	writeBufferSize int
}

type Config struct {
	serverAddress string
	session       SessionConfig
	upgrader      UpgraderConfig
}

func newConfig() *Config {
	config := &Config{
		serverAddress: defaultAddr,
		session:       SessionConfig{defaultWriteWait, defaultPongWait, defaultPingPeriod, defaultMaxMessageSize},
		upgrader:      UpgraderConfig{defaultReadBufferSize, defaultWriteBufferSize},
	}

	//var c ConfigFile
	//readFile(&c)
	//readEnv(&c)
	//
	//if len(c.Server.Addr) > 0 {
	//	config.serverAddress = c.Server.Addr
	//}

	return config
}

func readFile(config *ConfigFile) error {
	f, err := os.Open(defaultConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	return decoder.Decode(config)
}

func readEnv(config *ConfigFile) error {
	return envconfig.Process("", config)
}
