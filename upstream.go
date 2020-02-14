package main

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/smallnest/weighted"
)

type loadBalanceAlgorithm uint

const (
	roundRobing loadBalanceAlgorithm = iota
)

type loadBalancer interface {
	next() string
}

type upstreamConfig struct {
	name     string
	servers  []upstreamServer
	state    stringValue
	hash     upstreamHashConfig
	balancer loadBalancer
	once     sync.Once
}

type upstreamHashConfig struct {
	set        bool
	consistent boolValue
	key        string
}

func (u *upstreamConfig) algorithm() loadBalanceAlgorithm {
	var a loadBalanceAlgorithm
	return a
}

func (u *upstreamConfig) load(r *rule) error {
	if r.name != "upstream" {
		return errors.New("vince:not upstream server directive ")
	}
	u.name = r.args[0]
	for _, c := range r.children {
		switch c.name {
		case "server":
			var s upstreamServer
			if err := s.init(c); err != nil {
				return err
			}
			u.servers = append(u.servers, s)
		case "state":
			u.state.store(c.args[0])
		case "hash":
			u.hash.set = true
			u.hash.key = c.args[0]
			if len(c.args) > 0 && c.args[1] == "consistent" {
				u.hash.consistent.store(true)
			}
		}
	}
	return nil
}

func (s *upstreamServer) init(r *rule) error {
	s.url = r.args[0]
	for _, param := range r.args[1:] {
		switch param {
		case "backup":
			s.backup.store(true)
		case "down":
			s.down.store(true)
		case "resolve":
			s.resolve.store(true)
		case "drain":
			s.drain.store(true)
		default:
			parts := strings.Split(param, "=")
			switch parts[0] {
			case "weight":
				n, err := strconv.Atoi(parts[1])
				if err != nil {
					return err
				}
				s.weight.store(int64(n))
			case "max_conns":
				n, err := strconv.Atoi(parts[1])
				if err != nil {
					return err
				}
				s.maxConn.store(int64(n))
			case "max_fails":
				n, err := strconv.Atoi(parts[1])
				if err != nil {
					return err
				}
				s.maxConn.store(int64(n))

			case "fail_timeout":
				n, err := time.ParseDuration(parts[1])
				if err != nil {
					return err
				}
				s.failTimeout.store(n)
			case "route":
				s.route.store(parts[1])
			case "service":
				s.service.store(parts[1])
			case "slow_start":
				n, err := time.ParseDuration(parts[1])
				if err != nil {
					return err
				}
				s.slowStart.store(n)
			}
		}
	}
	return nil
}

func (u *upstreamConfig) init() {
	switch u.algorithm() {
	case roundRobing:
		w := &roundRobinWeighted{}
		for _, s := range u.servers {
			if s.down.value || s.backup.value {
				continue
			}
			w.add(s, int(s.weight.value))
		}
		u.balancer = w
	}
}

func (u *upstreamConfig) next() string {
	u.once.Do(u.init)
	return u.balancer.next()
}

type upstreamServer struct {
	url         string
	weight      intValue
	maxConn     intValue
	maxFails    intValue
	failTimeout durationValue
	backup      boolValue
	down        boolValue
	resolve     boolValue
	route       stringValue
	service     stringValue
	slowStart   durationValue
	drain       boolValue
}

type roundRobinWeighted struct {
	mu sync.Mutex
	rw *weighted.RRW
}

func (r *roundRobinWeighted) add(s interface{}, w int) {
	r.mu.Lock()
	r.rw.Add(s, w)
}

func (r *roundRobinWeighted) next() string {
	r.mu.Lock()
	nxt := r.rw.Next()
	r.mu.Unlock()
	return nxt.(string)
}

type roundRobinSmooth struct {
	mu sync.Mutex
	rw *weighted.SW
}

func (r *roundRobinSmooth) add(s string, w int) {
	r.mu.Lock()
	r.rw.Add(s, w)
}

func (r *roundRobinSmooth) next() string {
	r.mu.Lock()
	nxt := r.rw.Next()
	r.mu.Unlock()
	return nxt.(string)
}
