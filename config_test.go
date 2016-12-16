package taplink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCfgAppID(t *testing.T) {
	c := &Config{appID: "foobar"}
	assert.Equal(t, "foobar", c.AppID())
}

func TestCfgHost(t *testing.T) {
	c := &Config{host: "foobar"}
	assert.Equal(t, "foobar", c.Host())
}

func TestCfgHeaders(t *testing.T) {
	c := &Config{}
	assert.NotNil(t, c.Headers())
}

func TestCfgLastModified(t *testing.T) {
	c := &Config{}
	now := time.Now()
	now = time.Unix(now.Unix(), 0)
	assert.True(t, c.LastModified().IsZero())
	c.options = &Options{LastModified: now.Unix()}
	assert.Equal(t, now, c.LastModified())
}

func TestCfgServers(t *testing.T) {
	c := &Config{}
	assert.Len(t, c.Servers(), 0)
	c.options = &Options{Servers: []string{"foobar", "foobar2"}}
	assert.Equal(t, c.options.Servers, c.Servers())
}

func TestClientCfg(t *testing.T) {
	c := &Config{}
	client := HTTPClient
	assert.NotNil(t, c, client)
	assert.Equal(t, DefaultTimeout, client.Timeout)
}
