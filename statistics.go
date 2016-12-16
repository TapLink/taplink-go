package taplink

import "time"

// EnableStats enables the tracking of request statistics.
func (c *Client) EnableStats() {
	c.stats = true
}

// DisableStats disables the tracking of request statistics
func (c *Client) DisableStats() {
	c.stats = false
}

// Requests returns the number of requests sent to the API
func (c *Client) Requests() int64 {
	return c.reqCt
}

// Errors returns the number of requests send to the API which had errors
func (c *Client) Errors() int64 {
	return c.reqErrCt
}

// Latency returns the average latency of requests for successful requests
func (c *Client) Latency() time.Duration {
	c.RLock()
	defer c.RUnlock()
	var total time.Duration
	for i := range c.reqLatency {
		total += c.reqLatency[i]
	}
	return total / time.Duration(len(c.reqLatency))
}

// ErrorPct returns the percentage of requests which had errors
func (c *Client) ErrorPct() int64 {
	return c.Requests() / c.Errors()
}
