package main

import (
	"time"

	"github.com/uber-go/tally"
)

type metricsCollector struct {
	http struct {
		open         tally.Gauge
		active       tally.Gauge
		idle         tally.Gauge
		hijacked     tally.Gauge
		requestTotal tally.Counter
	}
	tcp struct {
		conn struct {
			accepted tally.Counter
			upstream struct {
			}
			handled tally.Counter
			reading tally.Gauge
			writing tally.Gauge
		}
		local struct {
			bytesRead    tally.Histogram
			bytesWritten tally.Histogram
		}
		upstream struct {
			bytesWritten tally.Histogram
			bytesRead    tally.Histogram
		}
		duration tally.Histogram
	}
}

func (m *metricsCollector) init(scope tally.Scope) {
	m.tcp.conn.accepted = scope.Counter("tcp_connections_accepted")
	m.tcp.conn.handled = scope.Counter("tcp_connections_handled")
	m.tcp.conn.reading = scope.Gauge("tcp_connections_reading")
	m.tcp.conn.writing = scope.Gauge("tcp_connections_writing")
	m.http.requestTotal = scope.Counter("http_requests_total")
	m.http.active = scope.Gauge("http_conn_active")
	m.http.idle = scope.Gauge("http_conn_idle")

	m.tcp.local.bytesRead = scope.Histogram("stream_local_bytes_read", histogramBucket())
	m.tcp.local.bytesWritten = scope.Histogram("stream_local_bytes_written", histogramBucket())
	m.tcp.upstream.bytesRead = scope.Histogram("stream_upstream_bytes_read", histogramBucket())
	m.tcp.upstream.bytesWritten = scope.Histogram("stream_upstream_bytes_written", histogramBucket())
	m.tcp.duration = scope.Histogram("stream_total_duration", tally.MustMakeLinearDurationBuckets(0, time.Millisecond, 60))
}

func (m *metricsCollector) reportHTTP(stats httpConnStatus) {
	m.http.open.Update(float64(stats.open.Load()))
	m.http.active.Update(float64(stats.active.Load()))
	m.http.idle.Update(float64(stats.hijacked.Load()))
}
