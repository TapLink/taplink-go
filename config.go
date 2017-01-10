package taplink

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

var (
	// Ensure the Config struct implements the Configuration interface
	_ Configuration = (*Config)(nil)

	userAgent = fmt.Sprintf("TapLink/1.0 Go/%s", goVersion)

	// DefaultHost is the default API host
	DefaultHost = "api.taplink.co"
)

// Configuration defines an interface which provides configuration info for requests to the API
type Configuration interface {
	AppID() string
	Host(attempts int) string
	Headers() map[string]string
	LastModified() time.Time
	Servers() []string
	Load() error

	Stats() Statistics
}

// Options is the options API response
type Options struct {
	LastModified int64    `json:"lastModified"`
	Servers      []string `json:"servers"`
}

// Config defines basic configuration for connecting to the API
type Config struct {
	appID     string
	headers   map[string]string
	options   *Options
	timeout   time.Duration
	keepAlive time.Duration
	client    API

	stats *statistics

	sync.RWMutex
}

// Load gets the configuration options from the API for the given app ID.
func (c *Config) Load() error {
	if c.options == nil {
		// c.Lock()
		c.options = &Options{Servers: make([]string, 0)}
		// c.Unlock()
	}
	resp, err := HTTPClient.Get(fmt.Sprintf("https://%s/%s", DefaultHost, c.appID))
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Could not get configuration: %v", err)
	}
	// c.Lock()
	// defer c.Unlock()
	if err := json.NewDecoder(resp.Body).Decode(c.options); err != nil {
		return err
	}
	// Init stats for each server.
	c.Stats().SetServers(c.options.Servers)
	return nil
}

// AppID returns the app ID
func (c *Config) AppID() string {
	return c.appID
}

// Stats returns a statistics interface for enabling/disabling/managing statistics.
func (c *Config) Stats() Statistics {
	c.Lock()
	defer c.Unlock()
	if c.stats == nil {
		c.stats = newStatistics()
	}
	return c.stats
}

type hostStats struct {
	host    string
	latency time.Duration
}

// Host returns the API server to connect to based on the available servers
// and the host selection algorithm
func (c *Config) Host(attempts int) string {

	hosts := c.Servers()
	if len(hosts) == 0 {
		return DefaultHost
	}
	// For the first attempt, return the first (primary) host
	if len(hosts) == 1 {
		return hosts[0]
	}
	return hosts[attempts%len(hosts)]
}

// Headers returns the headers to be added to each request
func (c *Config) Headers() map[string]string {
	if c.headers == nil {
		c.Lock()
		c.headers = make(map[string]string)
		c.Unlock()
	}
	return c.headers
}

// LastModified returns the last modification of the TapLink configuration
func (c *Config) LastModified() time.Time {
	c.RLock()
	defer c.RUnlock()
	if c.options != nil {
		return time.Unix(c.options.LastModified, 0)
	}
	return time.Time{}
}

// Servers returns the API servers available to connect to
func (c *Config) Servers() []string {
	c.RLock()
	defer c.RUnlock()
	if c.options == nil {
		return []string{}
	}
	return c.options.Servers
}
