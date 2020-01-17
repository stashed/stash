/*
Copyright The Stash Authors.

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

package backup

import (
	"regexp"
	"strings"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"

	ini "gopkg.in/ini.v1"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func sanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}

func sanitizeLabelValue(name string) string {
	return strings.Replace(name, "/", "|", -1)
}

func (c *Controller) JobName(resource *api.Restic) string {
	return sanitizeLabelValue(resource.Namespace + "-" + resource.Name)
}

func (c *Controller) GroupingKeys(resource *api.Restic) map[string]string {
	labels := make(map[string]string)
	labels[apis.LabelApp] = sanitizeLabelValue(c.opt.Workload.Name)
	labels["kind"] = sanitizeLabelValue(c.opt.Workload.Kind)
	labels["namespace"] = resource.Namespace
	labels["stash_config"] = resource.Name
	if cfg, err := ini.LooseLoad(c.opt.PodLabelsPath); err == nil {
		for _, section := range cfg.Sections() {
			for k, v := range section.KeysHash() {
				if k != "pod-template-hash" {
					labels["pod_"+sanitizeLabelName(k)] = sanitizeLabelValue(v)
				}
			}
		}
	}
	return labels
}
