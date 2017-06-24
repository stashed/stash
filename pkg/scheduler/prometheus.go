package scheduler

import (
	sapi "github.com/appscode/stash/api"
	_ "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	_ "github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/ini.v1"
)

func (c *controller) GroupingKeys(resource *sapi.Restic) map[string]string {
	labels := make(map[string]string)
	if c.opt.PrefixHostname {
		labels = push.HostnameGroupingKey()
	}
	labels["job"] = resource.Namespace + "/" + c.opt.Workload
	labels["namespace"] = resource.Namespace
	labels["stash_config"] = resource.Name
	if cfg, err := ini.LooseLoad(c.opt.PodLabelsPath); err == nil {
		for _, section := range cfg.Sections() {
			for k, v := range section.KeysHash() {
				labels["pod_"+k] = v
			}
		}
	}
	return labels
}
