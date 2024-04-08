/*
Copyright AppsCode Inc. and Contributors

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
	"fmt"
	"strings"
)

var ErrInvalidClusterManager = fmt.Errorf("not a valid ClusterManager, try [%s]", strings.Join(_ClusterManagerNames, ", "))

const _ClusterManagerName = "ACEOCMHubOCMMulticlusterControlplaneOCMSpokeOpenShiftRancherVirtualCluster"

var _ClusterManagerNames = []string{
	_ClusterManagerName[0:3],
	_ClusterManagerName[3:9],
	_ClusterManagerName[9:36],
	_ClusterManagerName[36:44],
	_ClusterManagerName[44:53],
	_ClusterManagerName[53:60],
	_ClusterManagerName[60:74],
}

// ClusterManagerNames returns a list of possible string values of ClusterManager.
func ClusterManagerNames() []string {
	tmp := make([]string, len(_ClusterManagerNames))
	copy(tmp, _ClusterManagerNames)
	return tmp
}

// ClusterManagerValues returns a list of the values for ClusterManager
func ClusterManagerValues() []ClusterManager {
	return []ClusterManager{
		ClusterManagerACE,
		ClusterManagerOCMHub,
		ClusterManagerOCMMulticlusterControlplane,
		ClusterManagerOCMSpoke,
		ClusterManagerOpenShift,
		ClusterManagerRancher,
		ClusterManagerVirtualCluster,
	}
}

var _ClusterManagerValue = map[string]ClusterManager{
	_ClusterManagerName[0:3]:   ClusterManagerACE,
	_ClusterManagerName[3:9]:   ClusterManagerOCMHub,
	_ClusterManagerName[9:36]:  ClusterManagerOCMMulticlusterControlplane,
	_ClusterManagerName[36:44]: ClusterManagerOCMSpoke,
	_ClusterManagerName[44:53]: ClusterManagerOpenShift,
	_ClusterManagerName[53:60]: ClusterManagerRancher,
	_ClusterManagerName[60:74]: ClusterManagerVirtualCluster,
}

// ParseClusterManager attempts to convert a string to a ClusterManager.
func ParseClusterManager(name string) (ClusterManager, error) {
	if x, ok := _ClusterManagerValue[name]; ok {
		return x, nil
	}
	return ClusterManager(0), fmt.Errorf("%s is %w", name, ErrInvalidClusterManager)
}
