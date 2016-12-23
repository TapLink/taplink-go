package taplink

import (
	"testing"

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
