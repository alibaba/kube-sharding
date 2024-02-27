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

	"github.com/prometheus/client_golang/prometheus"
	kmon_prometheus "github.com/prometheus/client_golang/prometheus"
)

type fakeMetricVec struct {
	delLvs map[string]bool
}

func (v *fakeMetricVec) DeleteLabelValues(lvs ...string) bool {
	v.delLvs[strings.Join(lvs, "/")] = true
	return true
}

func (v *fakeMetricVec) Describe(ch chan<- *kmon_prometheus.Desc) {
	ch <- kmon_prometheus.NewDesc("cpu_usage", "none", []string{"app", "role"}, kmon_prometheus.Labels{})
}

type prometheusFakeMetric struct {
	*fakeMetricVec
}

func (v *prometheusFakeMetric) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("cpu_usage", "none", []string{"app", "role"}, prometheus.Labels{})
}
