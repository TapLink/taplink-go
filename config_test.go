package taplink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	c := &Config{appID: testAppID}
	assert.NoError(t, c.Load())
}

func TestLoadInvalidApp(t *testing.T) {
	c := &Config{appID: "foobar"}
	assert.Error(t, c.Load())
	assert.NotNil(t, c.options)
}

func TestLoadMalformatted(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{200, 0, nil, []byte("foobar"), nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := &Config{appID: "foobar"}
	assert.Error(t, c.Load())
}

func TestCfgAppID(t *testing.T) {
	c := &Config{appID: "foobar"}
	assert.Equal(t, "foobar", c.AppID())
}

func TestCfgHost(t *testing.T) {
	c := &Config{}
	assert.Equal(t, DefaultHost, c.Host())
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

func TestConfigHost(t *testing.T) {
	c := &Config{options: &Options{Servers: []string{}}}
	defer func() {
		HostSelectionMethod = DefaultHostSelectionMethod
	}()

	// Test default host
	assert.Equal(t, DefaultHost, c.Host())

	// Test with only one host.
	c.options.Servers = []string{"foobar.com"}
	assert.Equal(t, "foobar.com", c.Host())

	// Test with multiple hosts
	c.options.Servers = []string{"foobar.com", "abc.foobar.com"}

	// First up, test random. Just make sure it doesn't panic since we
	// can't know the exact result other than if it's in bounds
	HostSelectionMethod = HostSelectRandom
	assert.NotPanics(t, func() {
		c.Host()
	})

	// Now test round robin
	HostSelectionMethod = HostSelectRoundRobin
	assert.Equal(t, 0, c.nextServer)
	assert.Equal(t, "foobar.com", c.Host())
	assert.Equal(t, 1, c.nextServer)
	assert.Equal(t, "abc.foobar.com", c.Host())
	assert.Equal(t, 0, c.nextServer)
	assert.Equal(t, "foobar.com", c.Host())
}
