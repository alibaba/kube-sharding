/*
Copyright 2024 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricgc

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	kmon_prometheus "github.com/prometheus/client_golang/prometheus"
	glog "k8s.io/klog"
)

type deletableMetricVec interface {
	DeleteLabelValues(lvs ...string) bool
}

type describableMetricVec interface {
	Describe() string
}

type metricVec interface {
	deletableMetricVec
	describableMetricVec
}

type prometheusMetricVec interface {
	deletableMetricVec
	Describe(chan<- *prometheus.Desc)
}

type prometheusKmonMetricVec interface {
	deletableMetricVec
	Describe(chan<- *kmon_prometheus.Desc)
}

type metricStats struct {
	sync.Mutex
	tm  time.Time
	lvs []string
	vec metricVec
}

func (s *metricStats) touch() {
	s.Lock()
	defer s.Unlock()
	s.tm = time.Now()
}

func (s *metricStats) check(duration time.Duration) bool {
	s.Lock()
	expired := time.Now().After(s.tm.Add(duration))
	s.Unlock()
	if expired {
		s.vec.DeleteLabelValues(s.lvs...)
	}
	return expired
}

type metricTracer struct {
	sync.RWMutex
	stats map[string]*metricStats
}

var tracer *metricTracer

func init() {
	tracer = &metricTracer{
		stats: make(map[string]*metricStats),
	}
}

func (t *metricTracer) getOrCreateStats(k string, lvs []string, vec metricVec) *metricStats {
	t.RLock()
	s, ok := t.stats[k]
	t.RUnlock()
	if ok {
		return s
	}
	t.Lock()
	defer t.Unlock()
	s, ok = t.stats[k]
	if !ok {
		s = &metricStats{
			lvs: lvs,
			vec: vec,
		}
		t.stats[k] = s
	}
	return s
}

func (t *metricTracer) start(checkDuration, expiredDuration time.Duration, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(checkDuration)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.check(expiredDuration)
			case <-stopCh:
				return
			}
		}
	}()
}

func (t *metricTracer) check(duration time.Duration) {
	t.Lock()
	defer t.Unlock()
	n := len(t.stats)
	for k, s := range t.stats {
		if s.check(duration) {
			glog.Infof("GC expired metric %s", k)
			delete(t.stats, k)
		}
	}
	if newn := len(t.stats); newn != n {
		glog.Infof("metricTracer traces total %d metrics, gc %d metrics", newn, n-newn)
	}
}

func (t *metricTracer) touchStats(vec metricVec, lvs ...string) {
	desc := vec.Describe()
	k := strings.Join(append([]string{desc}, lvs...), "/")
	stats := tracer.getOrCreateStats(k, lvs, vec)
	stats.touch()
}

// WithLabelValuesSummary touch a metric and return an Observer
func WithLabelValuesSummary(c *kmon_prometheus.SummaryVec, lvs ...string) kmon_prometheus.Observer {
	tracer.touchStats(newPrometheusMetricAdapter(c, nil), lvs...)
	return c.WithLabelValues(lvs...)
}

// WithLabelValuesSummary0 touch a metric and return an Observer
func WithLabelValuesSummary0(c *prometheus.SummaryVec, lvs ...string) prometheus.Observer {
	tracer.touchStats(newPrometheusMetricAdapter(nil, c), lvs...)
	return c.WithLabelValues(lvs...)
}

// WithLabelValuesCounter touch a metric and return a Counter
func WithLabelValuesCounter(c *kmon_prometheus.CounterVec, lvs ...string) kmon_prometheus.Counter {
	tracer.touchStats(newPrometheusMetricAdapter(c, nil), lvs...)
	return c.WithLabelValues(lvs...)
}

// WithLabelValuesCounter0 touch a metric and return a Counter
func WithLabelValuesCounter0(c *prometheus.CounterVec, lvs ...string) prometheus.Counter {
	tracer.touchStats(newPrometheusMetricAdapter(nil, c), lvs...)
	return c.WithLabelValues(lvs...)
}

// WithLabelValuesGauge touch a metric and return a Gauge
func WithLabelValuesGauge(c *kmon_prometheus.GaugeVec, lvs ...string) kmon_prometheus.Gauge {
	tracer.touchStats(newPrometheusMetricAdapter(c, nil), lvs...)
	return c.WithLabelValues(lvs...)
}

// WithLabelValuesGauge0 touch a metric and return a Gauge
func WithLabelValuesGauge0(c *prometheus.GaugeVec, lvs ...string) prometheus.Gauge {
	tracer.touchStats(newPrometheusMetricAdapter(nil, c), lvs...)
	return c.WithLabelValues(lvs...)
}

// Start starts metric tracer
func Start(stopCh <-chan struct{}) {
	tracer.start(time.Minute, time.Minute*5, stopCh)
}

var _ metricVec = &prometheusMetricAdapter{}

// prometheusMetricAdapter metric adapter
type prometheusMetricAdapter struct {
	prometheusKmonMetricVec prometheusKmonMetricVec
	prometheusMetricVec     prometheusMetricVec
}

func newPrometheusMetricAdapter(kmonVec prometheusKmonMetricVec, vec prometheusMetricVec) metricVec {
	return &prometheusMetricAdapter{
		prometheusKmonMetricVec: kmonVec,
		prometheusMetricVec:     vec,
	}
}

func (a *prometheusMetricAdapter) DeleteLabelValues(lvs ...string) bool {
	if a.prometheusKmonMetricVec != nil {
		return a.prometheusKmonMetricVec.DeleteLabelValues(lvs...)
	}
	if a.prometheusMetricVec != nil {
		return a.prometheusMetricVec.DeleteLabelValues(lvs...)
	}
	return false
}

func (a *prometheusMetricAdapter) describe(vec prometheusMetricVec) string {
	ch := make(chan *prometheus.Desc, 1)
	defer close(ch)
	vec.Describe(ch)
	desc := <-ch
	if desc != nil {
		return desc.String()
	}
	return ""
}

func (a *prometheusMetricAdapter) describeKmonMetricVec(vec prometheusKmonMetricVec) string {
	ch := make(chan *kmon_prometheus.Desc, 1)
	defer close(ch)
	vec.Describe(ch)
	desc := <-ch
	if desc != nil {
		return desc.String()
	}
	return ""
}

func (a *prometheusMetricAdapter) Describe() string {
	if a.prometheusKmonMetricVec != nil {
		return a.describeKmonMetricVec(a.prometheusKmonMetricVec)
	}
	if a.prometheusMetricVec != nil {
		return a.describe(a.prometheusMetricVec)
	}
	return ""
}
