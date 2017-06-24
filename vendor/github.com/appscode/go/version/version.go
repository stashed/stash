package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

type version struct {
	Version         string
	VersionStrategy string
	Os              string
	Arch            string
	CommitHash      string
	GitBranch       string
	GitTag          string
	CommitTimestamp string
	BuildTimestamp  string
	BuildHost       string
	BuildHostOs     string
	BuildHostArch   string
}

func (v *version) Print() {
	fmt.Printf("Version = %v\n", v.Version)
	fmt.Printf("VersionStrategy = %v\n", v.VersionStrategy)
	fmt.Printf("Os = %v\n", v.Os)
	fmt.Printf("Arch = %v\n", v.Arch)

	fmt.Printf("CommitHash = %v\n", v.CommitHash)
	fmt.Printf("GitBranch = %v\n", v.GitBranch)
	fmt.Printf("GitTag = %v\n", v.GitTag)
	if v.CommitTimestamp != "" {
		fmt.Printf("CommitTimestamp = %v\n", v.CommitTimestamp)
	}

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
		Run: func(cmd *cobra.Command, args []string) {
			Version.Print()
		},
	}
	return versionCmd
}
