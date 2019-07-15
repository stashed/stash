package context

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	yc "github.com/appscode/go/encoding/yaml"
	"github.com/appscode/go/log"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"gomodules.xyz/stow"
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
		log.Fatalf("error accessing flag osmconfig for command %s: %v", cmd.Name(), err)
	}
	return s
}

func LoadConfig(configPath string) (*OSMConfig, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}
	os.Chmod(configPath, 0600)

	config := &OSMConfig{}
	bytes, err := ioutil.ReadFile(configPath)
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
	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := ioutil.WriteFile(configPath, data, 0600); err != nil {
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
