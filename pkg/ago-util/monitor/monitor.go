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

package monitor

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Type Monitor type
type Type string

// The concrete monitor metric type
const (
	TypeGauge   = "gauge"
	TypeSummary = "summary"
	TypeCounter = "counter"
)

type reporter interface {
	Report(v float64, lvs ...string)
	Inc(lvs ...string)
}

type gaugeReporter struct {
	impl *prometheus.GaugeVec
}

func (r *gaugeReporter) Report(v float64, lvs ...string) {
	r.impl.WithLabelValues(lvs...).Set(v)
}

func (r *gaugeReporter) Inc(lvs ...string) {
	r.impl.WithLabelValues(lvs...).Inc()
}

type counterReporter struct {
	impl *prometheus.CounterVec
}

func (r *counterReporter) Report(v float64, lvs ...string) {
}

func (r *counterReporter) Inc(lvs ...string) {
	r.impl.WithLabelValues(lvs...).Inc()
}

type summaryReporter struct {
	impl *prometheus.SummaryVec
}

func (r *summaryReporter) Report(v float64, lvs ...string) {
	r.impl.WithLabelValues(lvs...).Observe(v)
}

func (r *summaryReporter) Inc(lvs ...string) {
}

var (
	registry map[string]reporter
)

func init() {
	registry = make(map[string]reporter)
}

// Config is the config to start monitor/metric.
type Config struct {
	Address string
	Port    int
	Service string
	Tags    map[string]string
}

// Register regists a metric.
func Register(t Type, fullname string, lbls ...string) error {
	var ns, sub, name string
	vec := strings.Split(fullname, ".")
	if len(vec) >= 3 {
		ns, sub, name = vec[0], vec[1], strings.Join(vec[2:], "_")
	} else if len(vec) == 2 {
		sub, name = vec[0], vec[1]
	} else {
		name = fullname
	}
	var reporter reporter
	if t == TypeGauge {
		impl := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      name,
			}, lbls)
		reporter = &gaugeReporter{impl}
		prometheus.MustRegister(impl)
	} else if t == TypeSummary {
		impl := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  ns,
				Subsystem:  sub,
				Name:       name,
				MaxAge:     time.Second * 30,
				AgeBuckets: 3,
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			}, lbls)
		reporter = &summaryReporter{impl}
		prometheus.MustRegister(impl)
	} else if t == TypeCounter {
		impl := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      name,
			}, lbls)
		reporter = &counterReporter{impl}
		prometheus.MustRegister(impl)
	}
	if reporter != nil {
		registry[fullname] = reporter
	}
	return nil
}

// Report reports a value for a metric.
func Report(fullname string, v float64, lvs ...string) {
	if reporter, ok := registry[fullname]; ok {
		reporter.Report(v, lvs...)
	}
}

// Inc reports a `inc` metric.
func Inc(fullname string, lvs ...string) {
	if reporter, ok := registry[fullname]; ok {
		reporter.Inc(lvs...)
	}
}

// ReportTime is a util function to report a `used time', e.g: defer ReportTime(xxx)()
func ReportTime(fullname string, lvs ...string) func() {
	t0 := nowInMillisecond()
	return func() {
		Report(fullname, float64(nowInMillisecond()-t0), lvs...)
	}
}

func nowInMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
