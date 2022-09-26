/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"strconv"

	"stash.appscode.dev/apimachinery/apis"

	core "k8s.io/api/core/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type VolumeTemplateOptions struct {
	Ordinal         int
	VolumeTemplates []ofst.PersistentVolumeClaim
}

func (opt VolumeTemplateOptions) Resolve() ([]core.PersistentVolumeClaim, error) {
	pvcList := make([]core.PersistentVolumeClaim, 0)
	for i := range opt.VolumeTemplates {
		inputs := make(map[string]string)
		inputs[apis.KeyPodOrdinal] = strconv.Itoa(opt.Ordinal)
		claim := opt.VolumeTemplates[i].DeepCopy().ToCorePVC()
		err := resolveWithInputs(claim, inputs)
		if err != nil {
			return nil, err
		}
		pvcList = append(pvcList, *claim)
	}
	return pvcList, nil
}
