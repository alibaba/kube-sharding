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

package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	marshalTimeout         = 3 * time.Second
	marshalConcurrentCount = 100
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CarbonJob CabronJob
type CarbonJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CarbonJobSpec   `json:"spec"`
	Status CarbonJobStatus `json:"status"`
}

// CarbonJobSpec spec for CarbonJobResource
type CarbonJobSpec struct {
	AppName  string         `json:"appName,omitempty"`
	Checksum string         `json:"checksum,omitempty"`
	AppPlan  *CarbonAppPlan `json:"appPlan,omitempty"`
	JobPlan  *CarbonJobPlan `json:"jobPlan,omitempty"`
}

type CarbonJobPlan struct {
	WorkerNodes map[string]WorkerNodeTemplate `json:"workerNodes,omitempty"`
}

type WorkerNodeTemplate struct {
	Immutable         bool `json:"immutable"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkerNodeSpec `json:"spec"`
}

// CarbonAppPlan appPlan for CarbonJob
type CarbonAppPlan struct {
	Groups map[string]*typespec.GroupPlan `json:"groups"`
}

// CarbonJobStatus status for carbon job
type CarbonJobStatus struct {
	SuccessCount int `json:"successCount"`
	DeadCount    int `json:"deadCount"`
	TotalCount   int `json:"totalCount"`
	TargetCount  int `json:"targetCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CarbonJobList CarbonJobList
type CarbonJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CarbonJob `json:"items"`
}

func (cb *CarbonJob) GetTaskSize() int {
	if cb.Spec.AppPlan != nil {
		return len(cb.Spec.AppPlan.Groups)
	}
	if cb.Spec.JobPlan != nil {
		return len(cb.Spec.JobPlan.WorkerNodes)
	}
	return 0
}

type carbonAppPlanJSONObj struct {
	Groups map[string]json.RawMessage `json:"groups"`
}

type carbonJobStatusJSONObj struct {
	Groups map[string]json.RawMessage `json:"groups"`
}

// MarshalJSON MarshalJSON
func (cap *CarbonAppPlan) MarshalJSON() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(marshalTimeout))
	defer cancel()

	fPlan := func(groupID string, groupPlan *typespec.GroupPlan) (string, json.RawMessage, error) {
		v, err := json.Marshal(groupPlan)
		if nil == err {
			return groupID, json.RawMessage(v), nil
		}
		return groupID, nil, err
	}

	batcher := utils.NewBatcher(marshalConcurrentCount)
	planCommands := make([]*utils.Command, 0, len(cap.Groups))
	for groupID, groupPlan := range cap.Groups {
		command, err := batcher.Go(
			ctx, false, fPlan, groupID, groupPlan,
		)
		if nil != err {
			return nil, err
		}
		planCommands = append(planCommands, command)
	}
	batcher.Wait()

	cbJSONObj := carbonAppPlanJSONObj{
		Groups: make(map[string]json.RawMessage, len(planCommands)),
	}
	errs := utils.NewErrors()
	for i := range planCommands {
		if nil != planCommands[i].FuncError {
			errs.Add(planCommands[i].FuncError)
			continue
		}
		if 3 != len(planCommands[i].Results) {
			return nil, fmt.Errorf("failed get plan marshal result, len errror")
		}
		if groupID, ok := planCommands[i].Results[0].(string); ok {
			if groupPlan, ok := planCommands[i].Results[1].(json.RawMessage); ok {
				cbJSONObj.Groups[groupID] = groupPlan
				continue
			}
		}
		errs.Add(fmt.Errorf("failed get plan marshal result"))
	}
	if nil != errs.Error() {
		return nil, errs.Error()
	}
	return json.Marshal(cbJSONObj)
}
