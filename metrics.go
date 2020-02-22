package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ergongate/vince/buffers"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/storage"
)

var (
	httpTotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "vince",
			Subsystem: "http",
			Name:      "total_requests",
		},
		[]string{"code", "method", "path"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "http",
			Name:      "request_duration",
		},
		[]string{"code", "method", "path"},
	)
	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "http",
			Name:      "request_size",
		},
		[]string{"code", "method", "path"},
	)
	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "http",
			Name:      "response_size",
		},
		[]string{"code", "method", "path"},
	)

	tcpLocalBytesRead = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "stream",
			Name:      "local_bytes_read",
		},
		[]string{"local_local", "local_remote", "remote_local", "remote_remote"},
	)
	tcpLocalBytesWritten = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "stream",
			Name:      "local_bytes_written",
		},
		[]string{"local_local", "local_remote", "remote_local", "remote_remote"},
	)
	tcpRemoteBytesRead = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "stream",
			Name:      "remote_bytes_read",
		},
		[]string{"local_local", "local_remote", "remote_local", "remote_remote"},
	)
	tcpRemoteBytesWritten = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "stream",
			Name:      "remote_bytes_written",
		},
		[]string{"local_local", "local_remote", "remote_local", "remote_remote"},
	)
	tcpStreamDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "vince",
			Subsystem: "stream",
			Name:      "stream_duration",
		},
		[]string{"local_local", "local_remote", "remote_local", "remote_remote"},
	)
	tcpTotalAcceptedConnection = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "vince",
			Subsystem: "stream",
			Name:      "accepted_connections",
		},
		[]string{"local_local", "local_remote"},
	)
)

func init() {
	prometheus.MustRegister(
		httpTotalRequests, httpRequestDuration, httpRequestSize, httpResponseSize,
		tcpLocalBytesRead, tcpLocalBytesWritten, tcpRemoteBytesRead, tcpRemoteBytesWritten,
	)
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}

type wrapResponseWriter struct {
	http.ResponseWriter
	code int
	size int64
}

func (w *wrapResponseWriter) WriteHeader(status int) {
	w.code = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *wrapResponseWriter) Write(b []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(b)
	w.size += int64(n)
	return
}

func instrumentEcho(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		duration := time.Since(start)
		code := strconv.Itoa(c.Response().Status)

		if err != nil {
			c.Error(err)
		}
		size := computeApproximateRequestSize(c.Request())

		httpRequestDuration.WithLabelValues(
			code, c.Request().Method, c.Path(),
		).Observe(float64(duration))
		httpRequestSize.WithLabelValues(
			code, c.Request().Method, c.Path(),
		).Observe(float64(size))
		httpResponseSize.WithLabelValues(
			code, c.Request().Method, c.Path(),
		).Observe(float64(c.Response().Size))
		httpTotalRequests.WithLabelValues(
			code, c.Request().Method, c.Path(),
		).Inc()
		return err
	}
}

func instrumentHandler(next handler) handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := &wrapResponseWriter{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(d, r)
		duration := time.Since(start)
		status := d.code
		if status == 0 {
			status = http.StatusOK
		}
		code := strconv.Itoa(status)
		size := computeApproximateRequestSize(r)
		httpRequestDuration.WithLabelValues(
			code, r.Method, r.URL.Path,
		).Observe(float64(duration))
		httpRequestSize.WithLabelValues(
			code, r.Method, r.URL.Path,
		).Observe(float64(size))
		httpResponseSize.WithLabelValues(
			code, r.Method, r.URL.Path,
		).Observe(float64(d.size))
		httpTotalRequests.WithLabelValues(
			code, r.Method, r.URL.Path,
		).Inc()
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	contentType := expfmt.Negotiate(r.Header)
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)
	enc := expfmt.NewEncoder(buf, contentType)
	var lastErr error
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if lastErr != nil && buf.Len() == 0 {
		http.Error(w, "No metrics encoded, last error:\n\n"+err.Error(), http.StatusInternalServerError)
		return
	}
	header := w.Header()
	header.Set(HeaderContentType, string(contentType))
	header.Set(HeaderContentLength, fmt.Sprint(buf.Len()))
	w.Write(buf.Bytes())
}

func storeMertics(app storage.Appender, f *dto.MetricFamily, ts time.Time) {
	defTime := timestamp.FromTime(ts)
	switch f.GetType() {
	case dto.MetricType_HISTOGRAM:
		for _, e := range histogramMetrics(f) {
			t := defTime
			if e.ts != 0 {
				t = e.ts
			}
			app.Add(e.labels, t, e.value)
		}
	case dto.MetricType_SUMMARY:
		for _, e := range summaryMetrics(f) {
			t := defTime
			if e.ts != 0 {
				t = e.ts
			}
			app.Add(e.labels, t, e.value)
		}
	default:
		for _, e := range basicMetrics(f) {
			t := defTime
			if e.ts != 0 {
				t = e.ts
			}
			app.Add(e.labels, t, e.value)
		}
	}
}

type metricEntry struct {
	labels labels.Labels
	ts     int64
	value  float64
}

func summaryMetrics(f *dto.MetricFamily) (o []metricEntry) {
	name := f.GetName()
	for _, m := range f.Metric {
		base := labels.NewBuilder(nil)
		ts := m.GetTimestampMs()
		for _, lp := range m.Label {
			base.Set(lp.GetName(), lp.GetValue())
		}
		s := m.GetSummary()
		if s != nil {
			o = append(o, metricEntryFor(name+"_sum", base.Labels(), ts, s.GetSampleSum()))
			o = append(o, metricEntryFor(name+"_count", base.Labels(), ts, float64(s.GetSampleCount())))
			for _, q := range s.GetQuantile() {
				o = append(o, metricEntryFor(name, append(base.Labels(), labels.Label{
					Name:  "quantile",
					Value: fmt.Sprint(q.GetQuantile()),
				}), ts, float64(q.GetValue())))
			}
		}
	}
	return
}

func histogramMetrics(f *dto.MetricFamily) (o []metricEntry) {
	name := f.GetName()
	for _, m := range f.Metric {
		base := labels.NewBuilder(nil)
		ts := m.GetTimestampMs()
		for _, lp := range m.Label {
			base.Set(lp.GetName(), lp.GetValue())
		}
		h := m.GetHistogram()
		if h != nil {
			o = append(o, metricEntryFor(name+"_sum", base.Labels(), ts, h.GetSampleSum()))
			o = append(o, metricEntryFor(name+"_count", base.Labels(), ts, float64(h.GetSampleCount())))
			for _, b := range h.GetBucket() {
				o = append(o, metricEntryFor(name+"_size_bucket", append(base.Labels(), labels.Label{
					Name:  "le",
					Value: fmt.Sprint(b.GetUpperBound()),
				}), ts, float64(b.GetCumulativeCount())))
			}
		}
	}
	return
}

func metricEntryFor(name string, base labels.Labels, ts int64, sum float64) metricEntry {
	var o metricEntry
	o.labels = append(labels.Labels{labels.Label{
		Name:  "__name__",
		Value: name,
	}}, base...)
	o.ts = ts
	o.value = sum
	return o
}

func basicMetrics(f *dto.MetricFamily) (o []metricEntry) {
	for _, m := range f.Metric {
		o = append(o, digestBasicMetric(*f.Name, m))
	}
	return
}

func digestBasicMetric(name string, m *dto.Metric) metricEntry {
	var o metricEntry
	o.labels = labels.Labels{labels.Label{
		Name:  "__name__",
		Value: name,
	}}
	for _, lp := range m.Label {
		o.labels = append(o.labels, labels.Label{
			Name:  *lp.Name,
			Value: *lp.Value,
		})
	}
	if m.TimestampMs != nil {
		o.ts = *m.TimestampMs
	}
	if m.Gauge != nil {
		o.value = m.Gauge.GetValue()
	}
	if m.Counter != nil {
		o.value = m.Counter.GetValue()
	}
	if m.Untyped != nil {
		o.value = m.Untyped.GetValue()
	}
	return o
}
