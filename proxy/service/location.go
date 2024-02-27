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

package service

type Location struct {
	Cluster      string      `json:"cluster,omitempty"`
	HippoAppName string      `json:"hippoAppName,omitempty"`
	Namespace    string      `json:"namespace,omitempty"`
	Rollingset   string      `json:"rollingset,omitempty"`
	Replica      string      `json:"replica,omitempty"`
	WorkerNode   string      `json:"workerNode,omitempty"`
	Serverless   *Serverless `json:"serverless,omitempty"`
}
type Serverless struct {
	AppName string `json:"appName,omitempty"`
	FiberId string `json:"fiberId,omitempty"`
}
