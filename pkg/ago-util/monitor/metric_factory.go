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

import "time"

// GenericMetric generic metric
type GenericMetric interface{}

// Counter metric (Gauge&Counter)
type Counter interface {
	Inc()
	Add(float64)
	Set(float64)
}

// Observer for Summary metric.
type Observer interface {
	Observe(float64)
}

// CurryMetric wraps generic metrics.
// Usage:
//   counterVec := prometheus.NewCounterVec(...)
//   factory.Register("count_metric_name", CurryMetric{func (lbls ...string) GenericMetric {
//       return counterVec.WithLabelValues(lbls...)
//   }})
type CurryMetric struct {
	Func func(lbls ...string) GenericMetric
}

// Inc increments the metric by 1.
func (c CurryMetric) Inc(lbls ...string) {
	if c.Func == nil {
		return
	}
	m := c.Func(lbls...)
	counter, ok := m.(Counter)
	if !ok {
		panic("Inc on invalid metric")
	}
	counter.Inc()
}

// Add increments the metric by `v`.
func (c CurryMetric) Add(v float64, lbls ...string) {
	if c.Func == nil {
		return
	}
	m := c.Func(lbls...)
	counter, ok := m.(Counter)
	if !ok {
		panic("Add on invalid metric")
	}
	counter.Add(v)
}

// Set set the metric value as `v`.
func (c CurryMetric) Set(v float64, lbls ...string) {
	if c.Func == nil {
		return
	}
	m := c.Func(lbls...)
	counter, ok := m.(Counter)
	if !ok {
		panic("Set on invalid metric")
	}
	counter.Set(v)
}

// Observe set the value.
func (c CurryMetric) Observe(f float64, lbls ...string) {
	if c.Func == nil {
		return
	}
	m := c.Func(lbls...)
	observer, ok := m.(Observer)
	if !ok {
		panic("Observe on invalid metric")
	}
	observer.Observe(f)
}

// Elapsed report an elapsed time in micro-seconds since `start`
func (c CurryMetric) Elapsed(start time.Time, lbls ...string) {
	c.Observe(float64(time.Since(start).Nanoseconds()/1000/1000), lbls...)
}

// MetricFactory provides api to get prometheus metrics by string name.
type MetricFactory struct {
	metrics map[string]CurryMetric
}

// NewMetricFactory constructs a new metric factory.
func NewMetricFactory() *MetricFactory {
	return &MetricFactory{metrics: make(map[string]CurryMetric)}
}

// Register registers a metric in this factory.
func (f *MetricFactory) Register(name string, m CurryMetric) {
	f.metrics[name] = m
}

// Must get a metric, panic if not existed.
func (f *MetricFactory) Must(name string) CurryMetric {
	m, ok := f.metrics[name]
	if !ok {
		panic("not found metric:" + name)
	}
	return m
}

// Get get a metric, return a dummy metric if not existed
func (f *MetricFactory) Get(name string) CurryMetric {
	m, ok := f.metrics[name]
	if !ok {
		return CurryMetric{}
	}
	return m
}
