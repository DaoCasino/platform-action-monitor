package monitor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

const testConfigFile = `
database:
  url: postgres://test:test@localhost/testCase
  filter:
    name: testName
    account: testAccount
server:
  addr: :31337
session:
  writeWait: 100s
  pongWait: 600s
  maxMessageSize: 2048
  maxEventsInMessage: 75
upgrader:
  readBufferSize: 1024
  writeBufferSize: 512
abi:
  main: contract_test.abi
  events:
    0: event_0.abi
    1: event_1.abi
eventExpires: 1 day
`

func TestConfigFile(t *testing.T) {
	reader := strings.NewReader(testConfigFile)
	configFile, err := newConfigFile(reader)
	require.NoError(t, err)

	assert.Equal(t, "postgres://test:test@localhost/testCase", configFile.Database.Url)
	assert.Equal(t, "testName", configFile.Database.Filter.Name)
	assert.Equal(t, "testAccount", configFile.Database.Filter.Account)

	assert.Equal(t, ":31337", configFile.Server.Addr)

	assert.Equal(t, "100s", configFile.Session.WriteWait)
	assert.Equal(t, "600s", configFile.Session.PongWait)
	assert.Equal(t, int64(2048), configFile.Session.MaxMessageSize)
	assert.Equal(t, 75, configFile.Session.MaxEventsInMessage)

	assert.Equal(t, 1024, configFile.Upgrader.ReadBufferSize)
	assert.Equal(t, 512, configFile.Upgrader.WriteBufferSize)

	assert.Equal(t, 2, len(configFile.Abi.Events))
	assert.Equal(t, "event_0.abi", configFile.Abi.Events[0])
	assert.Equal(t, "contract_test.abi", configFile.Abi.Main)

	assert.Equal(t, "1 day", configFile.EventExpires)
}

func TestConfigAssign(t *testing.T) {
	config := newConfig()

	reader := strings.NewReader(testConfigFile)
	configFile, err := newConfigFile(reader)
	require.NoError(t, err)

	err = config.assign(configFile)
	require.NoError(t, err)

	assert.Equal(t, "postgres://test:test@localhost/testCase", config.db.url)
	assert.Equal(t, "testName", *config.db.filter.actName)
	assert.Equal(t, "testAccount", *config.db.filter.actAccount)

	assert.Equal(t, ":31337", config.serverAddress)

	assert.Equal(t, 100*time.Second, config.session.writeWait)
	assert.Equal(t, 600*time.Second, config.session.pongWait)
	assert.Equal(t, int64(2048), config.session.maxMessageSize)
	assert.Equal(t, 75, config.session.maxEventsInMessage)

	assert.Equal(t, 1024, config.upgrader.readBufferSize)
	assert.Equal(t, 512, config.upgrader.writeBufferSize)

	assert.Equal(t, 2, len(config.abi.events))
	assert.Equal(t, "event_0.abi", config.abi.events[0])
	assert.Equal(t, "contract_test.abi", config.abi.main)

	assert.Equal(t, "1 day", config.eventExpires)

	configFile.Database.Filter.Name = ""
	configFile.Database.Filter.Account = ""

	err = config.assign(configFile)
	require.NoError(t, err)

	assert.Nil(t, config.db.filter.actName)
	assert.Nil(t, config.db.filter.actAccount)
}
