package taplink

import (
	"sync"
	"time"
)

var (
	_ Statistics = (*statistics)(nil)
)

// Statistics defines an interface for getting and setting connection statistics
type Statistics interface {
	Enable()
	Disable()
	AddLatency(host string, latency time.Duration)
	AddError(host string, code int)
	AddTimeout(host string)
	Get(host string) HostStats
}

type statistics struct {
	enabled bool
	stats   map[string]*hostStatistics

	mu sync.Mutex
}

func newStatistics() *statistics {
	return &statistics{stats: make(map[string]*hostStatistics)}
}

// Enable enables the tracking of request statistics.
func (s *statistics) Enable() {
	s.enabled = true
}

// Disable disables the tracking of request statistics
func (s *statistics) Disable() {
	s.enabled = false
}

func (s *statistics) AddLatency(host string, latency time.Duration) {
	if !s.enabled {
		return
	}
	s.init(host)
	l := append([]time.Duration(s.stats[host].latency), latency)
	s.stats[host].latency = Latency(l)
}

func (s *statistics) AddError(host string, code int) {
	if !s.enabled {
		return
	}
	s.init(host)
	s.stats[host].errors[code]++
}

func (s *statistics) AddTimeout(host string) {
	if !s.enabled {
		return
	}
	s.init(host)
	s.stats[host].timeouts++
}

func (s *statistics) Get(host string) HostStats {
	s.init(host)
	return s.stats[host]
}

func (s *statistics) init(host string) {
	if _, ok := s.stats[host]; !ok {
		s.stats[host] = newHostStatistics(host)
	}
}
