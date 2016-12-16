package taplink

import (
	"fmt"
	"time"
)

var (
	// Ensure the Config struct implements the Configuration interface
	_ Configuration = (*Config)(nil)

	userAgent = fmt.Sprintf("TapLink/1.0 Go/%s", goVersion)
)

// Configuration defines an interface which provides configuration info for requests to the API
type Configuration interface {
	AppID() string
	Host() string
	Headers() map[string]string
	LastModified() time.Time
	Servers() []string
}

// Options is the options API response
type Options struct {
	LastModified int64    `json:"lastModified"`
	Servers      []string `json:"servers"`
}

// Config defines basic configuration for connecting to the API
type Config struct {
	appID, host string
	headers     map[string]string
	options     *Options
	timeout     time.Duration
	keepAlive   time.Duration
}

// AppID returns the app ID
func (c *Config) AppID() string {
	return c.appID
}

// Host returns the API server to connect to
func (c *Config) Host() string {
	return c.host
}

// Headers returns the headers to be added to each request
func (c *Config) Headers() map[string]string {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	return c.headers
}

// LastModified returns the last modification of the TapLink configuration
func (c *Config) LastModified() time.Time {
	if c.options != nil {
		return time.Unix(c.options.LastModified, 0)
	}
	return time.Time{}
}

// Servers returns the API servers available to connect to
func (c *Config) Servers() []string {
	if c.options == nil {
		return []string{}
	}
	return c.options.Servers
}
