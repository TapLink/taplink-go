package taplink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHostStatisticsHost(t *testing.T) {
	s := &hostStatistics{host: "foobar.com"}
	assert.Equal(t, "foobar.com", s.Host())
}

func TestHostStatisticsTimeouts(t *testing.T) {
	c := New(testAppID).(*Client)
	c.Stats().AddTimeout("foobar.com")
	assert.Equal(t, int(0), c.Stats().Get("foobar.com").Timeouts())
	c.Stats().Enable()
	c.Stats().AddTimeout("foobar.com")
	assert.Equal(t, int(1), c.Stats().Get("foobar.com").Timeouts())
}

func TestHostStatisticsErrors(t *testing.T) {
	c := New(testAppID).(*Client)
	c.Stats().Enable()
	c.Stats().AddError("foobar.com", 503)
	c.Stats().AddError("foobar.com", 500)
	assert.Equal(t, 2, c.Stats().Get("foobar.com").Errors().Len())
	assert.Equal(t, 1, c.Stats().Get("foobar.com").Errors().Count(503))
	assert.Equal(t, 1, c.Stats().Get("foobar.com").Errors().Count(500))
	assert.Equal(t, 0, c.Stats().Get("foobar.com").Errors().Count(401))
}

func TestHostStatisticsLast(t *testing.T) {
	c := New(testAppID).(*Client)
	c.Stats().Enable()
	c.Stats().AddError("foobar.com", 503)
	c.Stats().AddSuccess("foobar.com", time.Millisecond)
	c.Stats().AddSuccess("foobar.com", time.Millisecond*3)
	c.Stats().AddTimeout("foobar.com")
	time.Sleep(2 * time.Second)
	c.Stats().AddError("foobar.com", 503)
	c.Stats().AddSuccess("foobar.com", time.Millisecond)
	c.Stats().AddTimeout("foobar.com")
	assert.Equal(t, int(3), c.Stats().Get("foobar.com").Latency().Len())
	assert.Equal(t, int(1), c.Stats().Get("foobar.com").Last(time.Second).Latency().Len())
	assert.Equal(t, int(2), c.Stats().Get("foobar.com").Errors().Len())
	assert.Equal(t, int(1), c.Stats().Get("foobar.com").Last(time.Second).Errors().Len())
	assert.Equal(t, int(2), c.Stats().Get("foobar.com").Timeouts())
	assert.Equal(t, int(1), c.Stats().Get("foobar.com").Last(time.Second).Timeouts())
	assert.Equal(t, float64(2)/float64(3), c.Stats().Get("foobar.com").Last(time.Second).ErrorRate())
	assert.Equal(t, float64(4)/float64(7), c.Stats().Get("foobar.com").ErrorRate())

}
