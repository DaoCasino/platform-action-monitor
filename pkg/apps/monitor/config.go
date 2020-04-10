package monitor

import (
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"time"
)

const (
	// TCP network address
	defaultAddr = ":8888"

	// Time allowed to write a message to the client.
	defaultWriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	defaultPongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	defaultPingPeriod = (defaultPongWait * 9) / 10

	// Maximum message size allowed from client.
	defaultMaxMessageSize = 1024 * 10

	// Maximum events to send per message
	defaultMaxEventsInMessage = 50

	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes in bytes. If a buffer
	// size is zero, then buffers allocated by the HTTP server are used. The
	// I/O buffer sizes do not limit the size of the messages that can be sent
	// or received.
	defaultReadBufferSize  = 1024
	defaultWriteBufferSize = 1024

	// path to files
	defaultContractABI = "../../../configs/abi/contract.abi"
	defaultEventABI    = "../../../configs/abi/event.abi"

	// pool_max_conns: integer greater than 0
	// pool_min_conns: integer 0 or greater
	// pool_max_conn_lifetime: duration string
	// pool_max_conn_idle_time: duration string
	// pool_health_check_period: duration string
	//
	//   # Example URL
	//   postgres://jack:secret@pg.example.com:5432/mydb?sslmode=verify-ca&pool_max_conns=10
	defaultDatabaseUrl = "postgres://test:test@localhost/test"

	// Life time event, use in fetchAllEvents SQL query block.timestamp > now() - interval '1 day'
	defaultEventExpires = "1 hour"
)

type SessionConfig struct {
	writeWait  time.Duration
	pongWait   time.Duration
	pingPeriod time.Duration

	maxMessageSize     int64
	maxEventsInMessage int
}

type UpgraderConfig struct {
	readBufferSize  int
	writeBufferSize int
}

type AbiConfig struct {
	main   string
	events map[int]string
}

type DatabaseFilters struct {
	actAccount *string
	actName    *string
}

type DatabaseConfig struct {
	url    string
	filter DatabaseFilters
}

type Config struct {
	db            DatabaseConfig
	serverAddress string
	session       SessionConfig
	upgrader      UpgraderConfig
	abi           AbiConfig
	eventExpires  string
}

type ConfigFile struct {
	Database struct {
		Url    string `yaml:"url"`
		Filter struct {
			Name    string `yaml:"name"`
			Account string `yaml:"account"`
		} `yaml:"filter"`
	} `yaml:"database"`

	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`

	Session struct {
		WriteWait      string `yaml:"writeWait"`
		PongWait       string `yaml:"pongWait"`
		MaxMessageSize int64  `yaml:"maxMessageSize"`
	} `yaml:"session"`

	Upgrader struct {
		ReadBufferSize  int `yaml:"readBufferSize"`
		WriteBufferSize int `yaml:"writeBufferSize"`
	} `yaml:"upgrader"`

	Abi struct {
		Main   string         `yaml:"main"`
		Events map[int]string `yaml:"events"`
	} `yaml:"abi"`

	EventExpires string `yaml:"eventExpires"`
}

func newDefaultConfig() *Config {
	config := &Config{
		db:            DatabaseConfig{defaultDatabaseUrl, DatabaseFilters{nil, nil}},
		serverAddress: defaultAddr,
		session:       SessionConfig{defaultWriteWait, defaultPongWait, defaultPingPeriod, defaultMaxMessageSize, defaultMaxEventsInMessage},
		upgrader:      UpgraderConfig{defaultReadBufferSize, defaultWriteBufferSize},
		abi:           AbiConfig{main: defaultContractABI, events: make(map[int]string)},
		eventExpires:  defaultEventExpires,
	}

	config.abi.events[0] = defaultEventABI

	return config
}

func (c *Config) assign(target *ConfigFile) (err error) {
	c.session.writeWait, err = time.ParseDuration(target.Session.WriteWait)
	if err != nil {
		return
	}
	c.session.pongWait, err = time.ParseDuration(target.Session.PongWait)
	if err != nil {
		return
	}
	c.session.pingPeriod = (c.session.pongWait * 9) / 10
	c.session.maxMessageSize = target.Session.MaxMessageSize

	c.db.url = target.Database.Url

	if target.Database.Filter.Name == "" {
		c.db.filter.actName = nil
	} else {
		c.db.filter.actName = &target.Database.Filter.Name
	}

	if target.Database.Filter.Account == "" {
		c.db.filter.actAccount = nil
	} else {
		c.db.filter.actAccount = &target.Database.Filter.Account
	}

	c.abi.main = target.Abi.Main
	c.abi.events = target.Abi.Events

	c.upgrader.writeBufferSize = target.Upgrader.WriteBufferSize
	c.upgrader.readBufferSize = target.Upgrader.ReadBufferSize

	c.serverAddress = target.Server.Addr

	if target.EventExpires != "" {
		c.eventExpires = target.EventExpires
	}
	return
}

func (c *Config) loadFromFile(filename *string) error {
	f, err := os.Open(*filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			mainLog.Error("loadFromFile error", zap.Error(err))
		}
	}()

	var configFile *ConfigFile
	configFile, err = newConfigFile(f)

	if err != nil {
		return err
	}

	return c.assign(configFile)
}

func newConfigFile(reader io.Reader) (*ConfigFile, error) {
	config := new(ConfigFile)
	decoder := yaml.NewDecoder(reader)
	err := decoder.Decode(config)
	return config, err
}

func newConfig() *Config {
	return newDefaultConfig()
}
