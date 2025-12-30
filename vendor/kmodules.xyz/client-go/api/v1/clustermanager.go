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
	"math/bits"
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

/*
ENUM(

	ACE                         = 1
	OCMHub                      = 2
	OCMMulticlusterControlplane = 4
	OCMSpoke                    = 8
	OpenShift                   = 16
	Rancher                     = 32
	VirtualCluster              = 64

)
*/
type ClusterManager int

const (
	ClusterManagerACE ClusterManager = 1 << iota
	ClusterManagerOCMHub
	ClusterManagerOCMMulticlusterControlplane
	ClusterManagerOCMSpoke
	ClusterManagerOpenShift
	ClusterManagerRancher
	ClusterManagerVirtualCluster
)

func (cm ClusterManager) ManagedByACE() bool {
	return cm&ClusterManagerACE == ClusterManagerACE
}

func (cm ClusterManager) ManagedByOCMHub() bool {
	return cm&ClusterManagerOCMHub == ClusterManagerOCMHub
}

func (cm ClusterManager) ManagedByOCMSpoke() bool {
	return cm&ClusterManagerOCMSpoke == ClusterManagerOCMSpoke
}

func (cm ClusterManager) ManagedByOCMMulticlusterControlplane() bool {
	return cm&ClusterManagerOCMMulticlusterControlplane == ClusterManagerOCMMulticlusterControlplane
}

func (cm ClusterManager) ManagedByRancher() bool {
	return cm&ClusterManagerRancher == ClusterManagerRancher
}

func (cm ClusterManager) ManagedByOpenShift() bool {
	return cm&ClusterManagerOpenShift == ClusterManagerOpenShift
}

func (cm ClusterManager) ManagedByVirtualCluster() bool {
	return cm&ClusterManagerVirtualCluster == ClusterManagerVirtualCluster
}

func (cm ClusterManager) Strings() []string {
	out := make([]string, 0, 7)
	if cm.ManagedByACE() {
		out = append(out, "ACE")
	}
	if cm.ManagedByOCMHub() {
		out = append(out, "OCMHub")
	}
	if cm.ManagedByOCMSpoke() {
		out = append(out, "OCMSpoke")
	}
	if cm.ManagedByOCMMulticlusterControlplane() {
		out = append(out, "OCMMulticlusterControlplane")
	}
	if cm.ManagedByRancher() {
		out = append(out, "Rancher")
	}
	if cm.ManagedByOpenShift() {
		out = append(out, "OpenShift")
	}
	if cm.ManagedByVirtualCluster() {
		out = append(out, "vcluster")
	}
	return out
}

func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

func (cm ClusterManager) Name() string {
	if !isPowerOfTwo(int(cm)) {
		return cm.String()
	}
	idx := bits.TrailingZeros(uint(cm))
	return _ClusterManagerNames[idx]
}

func (cm ClusterManager) String() string {
	return strings.Join(cm.Strings(), ",")
}
