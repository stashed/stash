package scheduler

import (
	sapi "github.com/appscode/stash/api"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/push"
)


func (c *controller) SetEnvVars2(resource *sapi.Restic) error {
	//push.Hos
}
