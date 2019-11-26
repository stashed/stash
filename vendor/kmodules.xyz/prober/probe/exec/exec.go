/*
Copyright 2015 The Kubernetes Authors.

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

package exec

import (
	"bytes"

	exec_util "kmodules.xyz/client-go/tools/exec"
	"kmodules.xyz/prober/api"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

const (
	maxReadLength = 10 * 1 << 10 // 10KB
)

// New creates a Prober.
func New() Prober {
	return execProber{}
}

// Prober is an interface defining the Probe object for container readiness/liveness checks.
type Prober interface {
	Probe(config *rest.Config, pod *core.Pod, containerName string, commands []string) (api.Result, string, error)
}

type execProber struct{}

// Probe executes a command to check the liveness/readiness of container
// from executing a command. Returns the Result status, command output, and
// errors if any.
func (pr execProber) Probe(config *rest.Config, pod *core.Pod, containerName string, commands []string) (api.Result, string, error) {
	// limit output and error msg size to 10KB
	var outBuffer, errBuffer bytes.Buffer
	stdOut := LimitWriter(&outBuffer, maxReadLength)
	stdErr := LimitWriter(&errBuffer, maxReadLength)

	container := containerName
	if container == "" {
		container = pod.Spec.Containers[0].Name
	}

	data, err := exec_util.ExecIntoPod(config, pod, func(opt *exec_util.Options) {
		opt.Container = container
		opt.Command = commands
		opt.StreamOptions.Stdout = stdOut
		opt.StreamOptions.Stderr = stdErr
	})

	if err != nil {
		return api.Failure, data, err
	}
	return api.Success, data, nil
}
