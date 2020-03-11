package main

import (
	"time"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`

	Session struct {
		WriteWait      time.Duration `yaml:"writeWait"`
		PongWait       time.Duration `yaml:"pongWait"`
		PingPeriod     time.Duration `yaml:"pingPeriod"`
		MaxMessageSize int64         `yaml:"maxMessageSize"`
	} `yaml:"session"`

	Upgrader struct {
		ReadBufferSize  int64 `yaml:"readBufferSize"`
		WriteBufferSize int64 `yaml:"writeBufferSize"`
	} `yaml:"upgrader"`
}
