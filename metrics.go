package main

import (
	"io"
	"time"

	"github.com/uber-go/tally"
	promreporter "github.com/uber-go/tally/prometheus"
)

type metricKey uint

const (
	httpOpenConnections metricKey = iota
	httpTotalRequests

	tcpAcceptedConn
	tcpLocalBytesRead
	tcpLocalBytesWrite
	tcpRemoteBytesRead
	tcpRemoteBytesWrite
	tcpTotalDuration
)

func (k metricKey) String() string {
	switch k {
	case httpTotalRequests:
		return "http_total_requests"
	case tcpAcceptedConn:
		return "tcp_accepted_conn"
	case tcpLocalBytesRead:
		return "tcp_local_bytes_read"
	case tcpLocalBytesWrite:
		return "tcp_local_bytes_Written"
	case tcpRemoteBytesRead:
		return "tcp_remote_bytes_read"
	case tcpRemoteBytesWrite:
		return "tcp_remote_bytes_Written"
	case tcpTotalDuration:
		return "tcp_total_duration"
	default:
		return "unknown"
	}
}

type collectorAPI interface {
	Counter(key metricKey, tags ...map[string]string) tally.Counter
	Gauge(key metricKey, tags ...map[string]string) tally.Gauge
	Timer(key metricKey, tags ...map[string]string) tally.Timer
}

type metricsCollector struct {
	scope    tally.Scope
	reporter tally.StatsReporter
	closer   io.Closer
}

func (m *metricsCollector) Close() error {
	return m.closer.Close()
}

func (m *metricsCollector) init() {
	r := promreporter.NewReporter(promreporter.Options{})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix: "vince",
		Tags: map[string]string{
			"version": "v0.1.0", //TODO: link version here
		},
		CachedReporter: r,
		Separator:      promreporter.DefaultSeparator,
	}, 1*time.Second)
	m.scope = scope
	m.closer = closer
}

func (m *metricsCollector) Counter(key metricKey, tags ...map[string]string) tally.Counter {
	if len(tags) > 0 {
		return m.scope.Tagged(tags[0]).Counter(key.String())
	}
	return m.scope.Counter(key.String())
}

func (m *metricsCollector) Gauge(key metricKey, tags ...map[string]string) tally.Gauge {
	if len(tags) > 0 {
		return m.scope.Tagged(tags[0]).Gauge(key.String())
	}
	return m.scope.Gauge(key.String())
}

func (m *metricsCollector) Timer(key metricKey, tags ...map[string]string) tally.Timer {
	if len(tags) > 0 {
		return m.scope.Tagged(tags[0]).Timer(key.String())
	}
	return m.scope.Timer(key.String())
}
