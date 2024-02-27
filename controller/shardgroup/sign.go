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

package shardgroup

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

func (c *Controller) calculateShardGroupSign(shardgroup *carbonv1.ShardGroup) (string, error) {
	shardgroupSpecBytes, err := json.Marshal(shardgroup.Spec)
	if err != nil {
		return "", err
	}
	shardgroupSignSpec := carbonv1.ShardGroupSignSpec{}
	err = json.Unmarshal(shardgroupSpecBytes, &shardgroupSignSpec)
	if err != nil {
		return "", err
	}

	shardgroupSign, err := utils.SignatureWithMD5(shardgroupSignSpec)
	if err != nil {
		return "", err
	}
	startIndex := strings.Index(shardgroup.Status.ShardGroupVersion, "-")
	if startIndex > 0 {
		curShardgroupSign := shardgroup.Status.ShardGroupVersion[startIndex+1:]
		if curShardgroupSign == shardgroupSign {
			return shardgroup.Status.ShardGroupVersion, nil
		}
	}
	return strconv.FormatInt(shardgroup.Generation, 10) + "-" + shardgroupSign, nil
}
