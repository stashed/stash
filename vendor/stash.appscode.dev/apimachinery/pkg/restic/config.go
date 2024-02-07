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

package restic

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	shell "gomodules.xyz/go-sh"
	core "k8s.io/api/core/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

const (
	DefaultOutputFileName = "output.json"
	DefaultScratchDir     = "/stash-tmp"
	DefaultHost           = "host-0"
)

type ResticWrapper struct {
	sh     *shell.Session
	config SetupOptions
}

type Command struct {
	Name string
	Args []interface{}
}

// BackupOptions specifies backup information
// if StdinPipeCommands is specified, BackupPaths will not be used
type BackupOptions struct {
	Host              string
	BackupPaths       []string
	StdinPipeCommands []Command
	StdinFileName     string // default "stdin"
	RetentionPolicy   v1alpha1.RetentionPolicy
	Exclude           []string
	Args              []string
}

// RestoreOptions specifies restore information
type RestoreOptions struct {
	Host         string
	SourceHost   string
	RestorePaths []string
	Snapshots    []string // when Snapshots are specified SourceHost and RestorePaths will not be used
	Destination  string   // destination path where snapshot will be restored, used in cli
	Exclude      []string
	Include      []string
	Args         []string
}

type DumpOptions struct {
	Host               string
	SourceHost         string
	Snapshot           string // default "latest"
	Path               string
	FileName           string // default "stdin"
	StdoutPipeCommands []Command
}

type SetupOptions struct {
	Provider       string
	Bucket         string
	Endpoint       string
	Region         string
	Path           string
	CacertFile     string
	ScratchDir     string
	EnableCache    bool
	InsecureTLS    bool
	MaxConnections int64
	StorageSecret  *core.Secret
	Nice           *ofst.NiceSettings
	IONice         *ofst.IONiceSettings
}

type KeyOptions struct {
	ID   string
	User string
	Host string
	File string
}

func NewResticWrapper(options SetupOptions) (*ResticWrapper, error) {
	wrapper := &ResticWrapper{
		sh:     shell.NewSession(),
		config: options,
	}

	err := wrapper.configure()
	if err != nil {
		return nil, err
	}
	return wrapper, nil
}

func NewResticWrapperFromShell(options SetupOptions, sh *shell.Session) (*ResticWrapper, error) {
	wrapper := &ResticWrapper{
		sh:     sh,
		config: options,
	}
	err := wrapper.configure()
	if err != nil {
		return nil, err
	}
	return wrapper, nil
}

func (w *ResticWrapper) configure() error {
	w.sh.SetDir(w.config.ScratchDir)
	w.sh.ShowCMD = true
	w.sh.PipeFail = true
	w.sh.PipeStdErrors = true

	// Setup restic environments
	return w.setupEnv()
}

func (w *ResticWrapper) SetEnv(key, value string) {
	if w.sh != nil {
		w.sh.SetEnv(key, value)
	}
}

func (w *ResticWrapper) GetEnv(key string) string {
	if w.sh != nil {
		return w.sh.Env[key]
	}
	return ""
}

func (w *ResticWrapper) GetCaPath() string {
	return w.config.CacertFile
}

func (w *ResticWrapper) DumpEnv(path string, dumpedFile string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	var envs string
	if w.sh != nil {
		sortedKeys := make([]string, 0, len(w.sh.Env))
		for k := range w.sh.Env {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys) // sort by key
		for _, v := range sortedKeys {
			envs = envs + fmt.Sprintln(v+"="+w.sh.Env[v])
		}
	}

	if err := os.WriteFile(filepath.Join(path, dumpedFile), []byte(envs), 0o600); err != nil {
		return err
	}
	return nil
}

func (w *ResticWrapper) HideCMD() {
	if w.sh != nil {
		w.sh.ShowCMD = false
	}
}

func (w *ResticWrapper) GetRepo() string {
	if w.sh != nil {
		return w.sh.Env[RESTIC_REPOSITORY]
	}
	return ""
}

// Copy function copy input ResticWrapper and returns a new wrapper with copy of its content.
func (w *ResticWrapper) Copy() *ResticWrapper {
	if w == nil {
		return nil
	}
	out := new(ResticWrapper)

	if w.sh != nil {
		out.sh = shell.NewSession()

		// set values in.sh to out.sh
		for k, v := range w.sh.Env {
			out.sh.Env[k] = v
		}
		// don't use same stdin, stdout, stderr for each instant to avoid data race.
		// out.sh.Stdin = in.sh.Stdin
		// out.sh.Stdout = in.sh.Stdout
		// out.sh.Stderr = in.sh.Stderr
		out.sh.ShowCMD = w.sh.ShowCMD
		out.sh.PipeFail = w.sh.PipeFail
		out.sh.PipeStdErrors = w.sh.PipeStdErrors

	}
	out.config = w.config
	return out
}
