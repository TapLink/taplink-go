package taplink

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLatency(t *testing.T) {
	c := New(testAppID)
	c.Stats().Enable()
	c.Stats().AddSuccess("foobar.com", 10*time.Millisecond)
	c.Stats().AddSuccess("foobar.com", 20*time.Millisecond)
	c.Stats().AddSuccess("foobar.com", 30*time.Millisecond)
	c.Stats().AddSuccess("foobar.com", 40*time.Millisecond)
	c.Stats().AddSuccess("foobar.com", 50*time.Millisecond)
	assert.Equal(t, 30*time.Millisecond, c.Stats().Get("foobar.com").Latency().Avg())
}

func TestStatsGetNil(t *testing.T) {
	c := New(testAppID)
	assert.NotPanics(t, func() {
		c.Stats().Get("foobar")
	})
	assert.NotPanics(t, func() {
		c.Stats().(*statistics).stats = nil
		c.Stats().(*statistics).init("foobar.com")
	})
}

func TestStatsEnabled(t *testing.T) {
	s := &statistics{}
	s.Enable()
	assert.True(t, s.enabled)
	s.Disable()
	assert.False(t, s.enabled)
}

func TestHostSorting(t *testing.T) {
	// foo.com will have errors, bar.com will not, so bar.com should be the server of choice
	f := newHostStatistics("foo.com")
	b := newHostStatistics("bar.com")
	f.errors = []errorResp{{time.Now(), 503}}
	b.latency = []successResp{{time.Now(), time.Millisecond}}
	l := hostFailRate([]hostStatistics{f.CopyOf(), b.CopyOf()})
	sort.Sort(l)
	assert.Equal(t, []string{"bar.com", "foo.com"}, l.Hosts())

	// Set the servers to some test values, add stats, and make sure the proper one is selected.
	// Usually the config would be loaded from c.Config().Load() but so we can test we'll set it
	// manually here.
	svrs := []string{"foo.com", "bar.com", "foobar.com"}
	c := New(testAppID)
	c.Config().Load()
	c.(*Client).cfg.(*Config).options.Servers = svrs
	c.Stats().(*statistics).stats = map[string]*hostStatistics{
		"foo.com":    newHostStatistics("foo.com"),
		"bar.com":    newHostStatistics("bar.com"),
		"foobar.com": newHostStatistics("foobar.com"),
	}

	assert.Equal(t, []string{"foo.com", "bar.com", "foobar.com"}, c.Stats().Hosts())
	c.Stats().AddError("foo.com", 503)
	c.Stats().AddSuccess("foo.com", time.Millisecond)
	c.Stats().AddSuccess("bar.com", time.Millisecond)

	// The preferred host should be attempted first.
	assert.Equal(t, "foo.com", c.Config().Host(0))

	// After than, the sorted list, where we expect bar.com to be first because it has no errors,
	// then foobar.com, because it has no errors, then foo.com because it has errors.
	assert.Equal(t, "bar.com", c.Config().Host(1))
	assert.Equal(t, "foobar.com", c.Config().Host(2))
	assert.Equal(t, "foo.com", c.Config().Host(3))
}
