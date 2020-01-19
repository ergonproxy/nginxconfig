package main

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

type limiter interface {
	take() bool
}

type leaky struct {
	capacity  int
	remaining int
	reset     time.Time
	rate      time.Duration
	amount    int
	mutex     sync.Mutex
}

func (l *leaky) take() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if time.Now().After(l.reset) {
		l.reset = time.Now().Add(l.rate)
		l.remaining = l.capacity
	}
	if l.amount > l.remaining {
		return false
	}
	l.remaining -= l.amount
	return true
}

type limitReqOpts struct {
	key  string
	zone string
	rate int
}

type rateLimiterManager struct {
	zones *sync.Map
}

func newRateManager() *rateLimiterManager {
	return &rateLimiterManager{zones: new(sync.Map)}
}

func (r *rateLimiterManager) setup(key, zone string, rate int) limiter {
	if z, ok := r.zones.Load(zone); ok {
		if l, ok := z.(*sync.Map).Load(key); ok {
			return l.(limiter)
		}
		lmt := &leaky{
			capacity:  rate,
			remaining: rate,
			reset:     time.Now().Add(time.Second),
			rate:      time.Second,
		}
		z.(*sync.Map).Store(key, lmt)
		return lmt
	}
	z := new(sync.Map)
	lmt := &leaky{}
	z.Store(key, lmt)
	r.zones.Store(zone, z)
	return lmt
}

// parses rate that is defined in nginx format eg 1r/s 10r/m and return the
// number of request per second.
func parseRate(s string) (int, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, errors.New("vince: invalid rate value")
	}
	r := strings.TrimSuffix(parts[0], "r")
	if parts[1] == "s" {
		return strconv.Atoi(r)
	}
	duration, err := time.ParseDuration("1" + parts[1])
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(r)
	if err != nil {
		return 0, err
	}
	return int(v / int(duration.Seconds())), nil
}
