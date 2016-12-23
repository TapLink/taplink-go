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
}

type hostStatistics struct {
	timeouts int
	errors   map[int]int
	host     string
	latency  []time.Duration

	mu sync.RWMutex
}

func newHostStatistics(host string) *hostStatistics {
	return &hostStatistics{
		host:     host,
		errors:   Errors(make(map[int]int, 0)),
		latency:  Latency(make([]time.Duration, 0)),
		timeouts: 0,
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
	return Errors(s.errors)
}

func (s *hostStatistics) Requests() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.latency)
}

func (s *hostStatistics) Latency() Latency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Latency(s.latency)
}

func (s *hostStatistics) Timeouts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.timeouts
}
