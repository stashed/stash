package version

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

type version struct {
	Version         string `json:"version,omitempty"`
	VersionStrategy string `json:"versionStrategy,omitempty"`
	CommitHash      string `json:"commitHash,omitempty"`
	GitBranch       string `json:"gitBranch,omitempty"`
	GitTag          string `json:"gitTag,omitempty"`
	CommitTimestamp string `json:"commitTimestamp,omitempty"`
	GoVersion       string `json:"goVersion,omitempty"`
	Compiler        string `json:"compiler,omitempty"`
	Platform        string `json:"platform,omitempty"`
}

func (v *version) Print() {
	fmt.Printf("Version = %v\n", v.Version)
	fmt.Printf("VersionStrategy = %v\n", v.VersionStrategy)
	fmt.Printf("GitTag = %v\n", v.GitTag)
	fmt.Printf("GitBranch = %v\n", v.GitBranch)
	fmt.Printf("CommitHash = %v\n", v.CommitHash)
	fmt.Printf("CommitTimestamp = %v\n", v.CommitTimestamp)

	if v.GoVersion != "" {
		fmt.Printf("GoVersion = %v\n", v.GoVersion)
	}
	if v.Compiler != "" {
		fmt.Printf("Compiler = %v\n", v.Compiler)
	}
	if v.Platform != "" {
		fmt.Printf("Platform = %v\n", v.Platform)
	}
}

var Version version

func NewCmdVersion() *cobra.Command {
	var short bool
	var check string
	cmd := &cobra.Command{
		Use:               "version",
		Short:             "Prints binary version number.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if short {
				fmt.Print(Version.Version)
			} else {
				Version.Print()
			}
			if check != "" {
				c, err := semver.NewConstraint(check)
				if err != nil {
					return fmt.Errorf("failed to parse --check: %v", err)
				}
				v, err := semver.NewVersion(Version.Version)
				if err != nil {
					return fmt.Errorf("failed to parse version: %v", err)
				}
				if !c.Check(v) {
					return fmt.Errorf("version %q fails to meet constraint %q", v.String(), c.String())
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "Print just the version number.")
	cmd.Flags().StringVar(&check, "check", "", "Check version constraint")
	return cmd
}
