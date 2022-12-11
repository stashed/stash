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

package osm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	yc "gomodules.xyz/encoding/yaml"
	"gomodules.xyz/stow"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type Context struct {
	Name     string         `json:"name"`
	Provider string         `json:"provider"`
	Config   stow.ConfigMap `json:"config"`
}

type OSMConfig struct {
	Contexts       []*Context `json:"contexts"`
	CurrentContext string     `json:"current-context"`
}

func GetConfigPath(cmd *cobra.Command) string {
	s, err := cmd.Flags().GetString("osmconfig")
	if err != nil {
		klog.Fatalf("error accessing flag osmconfig for command %s: %v", cmd.Name(), err)
	}
	return s
}

func LoadConfig(configPath string) (*OSMConfig, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}
	err := os.Chmod(configPath, 0o600)
	if err != nil {
		return nil, err
	}

	config := &OSMConfig{}
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	jsonData, err := yc.ToJSON(bytes)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonData, config)
	return config, err
}

func (config *OSMConfig) Save(configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(configPath), 0o755)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return err
	}
	return nil
}

func (config *OSMConfig) Dial(cliCtx string) (stow.Location, error) {
	ctx := config.CurrentContext
	if cliCtx != "" {
		ctx = cliCtx
	}
	for _, osmCtx := range config.Contexts {
		if osmCtx.Name == ctx {
			return stow.Dial(osmCtx.Provider, osmCtx.Config)
		}
	}
	return nil, errors.New("failed to determine context")
}

func (config *OSMConfig) Context(cliCtx string) (*Context, error) {
	ctx := config.CurrentContext
	if cliCtx != "" {
		ctx = cliCtx
	}
	for _, osmCtx := range config.Contexts {
		if osmCtx.Name == ctx {
			return osmCtx, nil
		}
	}
	return nil, errors.New("failed to determine context")
}
