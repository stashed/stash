package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

type version struct {
	Version         string `json:"version,omitempty"`
	VersionStrategy string `json:"versionStrategy,omitempty"`
	Os              string `json:"os,omitempty"`
	Arch            string `json:"arch,omitempty"`
	CommitHash      string `json:"commitHash,omitempty"`
	GitBranch       string `json:"gitBranch,omitempty"`
	GitTag          string `json:"gitTag,omitempty"`
	CommitTimestamp string `json:"commitTimestamp,omitempty"`
	BuildTimestamp  string `json:"buildTimestamp,omitempty"`
	BuildHost       string `json:"buildHost,omitempty"`
	BuildHostOs     string `json:"buildHostOs,omitempty"`
	BuildHostArch   string `json:"buildHostArch,omitempty"`
}

func (v *version) Print() {
	fmt.Printf("Version = %v\n", v.Version)
	fmt.Printf("VersionStrategy = %v\n", v.VersionStrategy)
	fmt.Printf("Os = %v\n", v.Os)
	fmt.Printf("Arch = %v\n", v.Arch)

	fmt.Printf("CommitHash = %v\n", v.CommitHash)
	fmt.Printf("GitBranch = %v\n", v.GitBranch)
	fmt.Printf("GitTag = %v\n", v.GitTag)
	fmt.Printf("CommitTimestamp = %v\n", v.CommitTimestamp)

	if v.BuildTimestamp != "" {
		fmt.Printf("BuildTimestamp = %v\n", v.BuildTimestamp)
	}
	if v.BuildHost != "" {
		fmt.Printf("BuildHost = %v\n", v.BuildHost)
	}
	if v.BuildHostOs != "" {
		fmt.Printf("BuildHostOs = %v\n", v.BuildHostOs)
	}
	if v.BuildHostArch != "" {
		fmt.Printf("BuildHostArch = %v\n", v.BuildHostArch)
	}
}

var Version version

func NewCmdVersion() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints binary version number.",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			Version.Print()
		},
	}
	return versionCmd
}
