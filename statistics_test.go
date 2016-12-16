package taplink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLatency(t *testing.T) {
	c := &Client{}
	c.reqLatency = []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}
	assert.Equal(t, 30*time.Millisecond, c.Latency())
}

func TestErrorPct(t *testing.T) {
	c := &Client{}
	c.reqCt = 100
	c.reqErrCt = 10
	assert.Equal(t, int64(10), c.ErrorPct())
}

func TestEnableStats(t *testing.T) {
	c := &Client{}
	c.EnableStats()
	assert.True(t, c.stats)
}

func TestDisableStatus(t *testing.T) {
	c := &Client{}
	c.DisableStats()
	assert.False(t, c.stats)
}
