package scheduler

import (
	"os"
	"regexp"
	"strings"

	sapi "github.com/appscode/stash/api"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/ini.v1"
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

func (c *Scheduler) JobName(resource *sapi.Restic) string {
	if c.opt.App != "" {
		return sanitizeLabelValue(resource.Namespace + "-" + c.opt.App)
	}
	if host, err := os.Hostname(); err != nil {
		return sanitizeLabelValue(resource.Namespace + "-" + host)
	}
	return ""
}

func (c *Scheduler) GroupingKeys(resource *sapi.Restic) map[string]string {
	labels := make(map[string]string)
	if c.opt.PrefixHostname {
		labels = push.HostnameGroupingKey()
	}
	if c.opt.App != "" {
		labels["app"] = sanitizeLabelValue(c.opt.App)
	}
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
