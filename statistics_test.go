package taplink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLatency(t *testing.T) {
	c := &Client{}
	c.Stats().Enable()
	c.Stats().AddLatency("foobar.com", 10*time.Millisecond)
	c.Stats().AddLatency("foobar.com", 20*time.Millisecond)
	c.Stats().AddLatency("foobar.com", 30*time.Millisecond)
	c.Stats().AddLatency("foobar.com", 40*time.Millisecond)
	c.Stats().AddLatency("foobar.com", 50*time.Millisecond)
	assert.Equal(t, 30*time.Millisecond, c.Stats().Get("foobar.com").Latency().Avg())
}

func TestStatsGetNil(t *testing.T) {
	c := &Client{}
	assert.NotPanics(t, func() {
		c.Stats().Get("foobar")
	})
}

func TestStatsEnabled(t *testing.T) {
	s := &statistics{}
	s.Enable()
	assert.True(t, s.enabled)
	s.Disable()
	assert.False(t, s.enabled)
}
