package taplink

import (
	"sync"
	"time"
)

var (
	_ HostStats  = (*hostStatistics)(nil)
	_ Statistics = (*statistics)(nil)
)

// Latency is a slice of duration of the requests.
type Latency []time.Duration

// Avg returns the average latency for the slice
func (l Latency) Avg() time.Duration {
	if len(l) == 0 {
		return 0
	}
	var total time.Duration
	for i := range l {
		total += l[i]
	}
	return total / time.Duration(len(l))
}

// Len returns the length of the underlying slice
func (l Latency) Len() int {
	return len([]time.Duration(l))
}

// Errors is a map of how error codes (key) and count of those codes (value)
type Errors map[int]int

// Len returns the total number of errors
func (e Errors) Len() (l int) {
	for i := range e {
		l += e[i]
	}
	return
}

// Count returns the number of errors for the given code.
func (e Errors) Count(code int) int {
	for i, ct := range e {
		if code == i {
			return ct
		}
	}
	return 0
}

// HostStats defines an interface which provides detailed information about the
// statistics related to connections to the given host.
type HostStats interface {
	Errors() Errors
	Requests() int
	Timeouts() int
	Latency() Latency
	ErrorRate() float64
	Last(time.Duration) HostStats
}

type errorResp struct {
	ts   time.Time
	code int
}

type successResp struct {
	ts      time.Time
	latency time.Duration
}

type timeoutResp struct {
	ts time.Time
}

type hostStatistics struct {
	errors   []errorResp
	timeouts []timeoutResp
	latency  []successResp
	host     string

	mu sync.RWMutex
}

func newHostStatistics(host string) *hostStatistics {
	return &hostStatistics{
		host:     host,
		errors:   make([]errorResp, 0),
		latency:  make([]successResp, 0),
		timeouts: make([]timeoutResp, 0),
	}
}

func (s *hostStatistics) Host() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.host
}

func (s *hostStatistics) Errors() Errors {
	s.mu.Lock()
	defer s.mu.Unlock()
	errs := make(map[int]int, 0)
	for i := range s.errors {
		errs[s.errors[i].code]++
	}
	return Errors(errs)
}

func (s *hostStatistics) Requests() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.latency)
}

func (s *hostStatistics) Latency() Latency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lat := make([]time.Duration, len(s.latency))
	for i := range s.latency {
		lat[i] = s.latency[i].latency
	}
	return Latency(lat)
}

func (s *hostStatistics) Timeouts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.timeouts)
}

func (s *hostStatistics) ErrorRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	errCt := len(s.timeouts) + len(s.errors)
	totalCt := len(s.latency) + len(s.timeouts) + len(s.errors)
	if errCt == 0 {
		return 0
	}
	return float64(errCt) / float64(totalCt)
}

// Since returns a subset of the host statistics for events which happend between now and since.
func (s *hostStatistics) Last(last time.Duration) HostStats {

	s.mu.RLock()
	lat := s.latency
	errs := s.errors
	tos := s.timeouts
	s.mu.RUnlock()

	var om hostStatistics
	if last > 0 {
		last *= -1
	}
	u := time.Now().Add(last)
	for i := range lat {
		if s.latency[i].ts.Before(u) {
			continue
		}
		om.latency = append(om.latency, lat[i])
	}

	for i := range errs {
		if s.errors[i].ts.Before(u) {
			continue
		}
		om.errors = append(om.errors, errs[i])
	}

	for i := range tos {
		if s.timeouts[i].ts.Before(u) {
			continue
		}
		om.timeouts = append(om.timeouts, tos[i])
	}

	return &om
}
