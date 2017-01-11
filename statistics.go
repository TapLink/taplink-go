package taplink

import (
	"sort"
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
	AddSuccess(host string, latency time.Duration)
	AddError(host string, code int)
	AddTimeout(host string)
	Get(host string) HostStats
	SetServers(servers []string)
	Hosts() []string
}

type statistics struct {
	enabled bool
	stats   map[string]*hostStatistics

	mu sync.RWMutex
}

func newStatistics() *statistics {
	return &statistics{stats: make(map[string]*hostStatistics)}
}

// Enable enables the tracking of request statistics.
func (s *statistics) Enable() {
	s.mu.Lock()
	s.enabled = true
	s.mu.Unlock()
}

// Disable disables the tracking of request statistics
func (s *statistics) Disable() {
	s.mu.Lock()
	s.enabled = false
	s.mu.Unlock()
}

func (s *statistics) AddSuccess(host string, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return
	}
	s.init(host)
	s.stats[host].latency = append(s.stats[host].latency, successResp{time.Now(), latency})
}

func (s *statistics) AddError(host string, code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return
	}
	s.init(host)
	s.stats[host].errors = append(s.stats[host].errors, errorResp{time.Now(), code})
}

func (s *statistics) AddTimeout(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return
	}
	s.init(host)
	s.stats[host].timeouts = append(s.stats[host].timeouts, timeoutResp{time.Now()})
}

func (s *statistics) Get(host string) HostStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init(host)
	return s.stats[host]
}

// SetServers initializes statistics for the given servers
func (s *statistics) SetServers(servers []string) {
	for i := range servers {
		s.init(servers[i])
	}
}

type hostFailRate []hostStatistics

func (hfr hostFailRate) Len() int { return len(hfr) }

func (hfr hostFailRate) Swap(i, j int) { hfr[i], hfr[j] = hfr[j].CopyOf(), hfr[i].CopyOf() }

func (hfr hostFailRate) Less(i, j int) bool {
	im := hfr[i].Last(time.Minute)
	jm := hfr[j].Last(time.Minute)
	return im.ErrorRate() < jm.ErrorRate() || im.Latency().Avg() < jm.Latency().Avg()
}

func (hfr hostFailRate) Hosts() []string {
	hosts := make([]string, len(hfr))
	for i := range hfr {
		hosts[i] = hfr[i].Host()
	}
	return hosts
}

// Hosts returns a sorted slice of hosts, with the most optimal host being first.
// Hosts are sorted by error rate and if error rate is equal, then latency.
func (s *statistics) Hosts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := make([]hostStatistics, 0)
	for h := range s.stats {
		l = append(l, s.stats[h].CopyOf())
	}
	hfr := hostFailRate(l)
	sort.Sort(hfr)
	return hfr.Hosts()
}

func (s *statistics) init(host string) {
	if s.stats == nil {
		s.stats = make(map[string]*hostStatistics, 0)
	}
	if _, ok := s.stats[host]; !ok {
		s.stats[host] = newHostStatistics(host)
	}
}
