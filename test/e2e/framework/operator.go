package framework

import (
	"path/filepath"

	srvr "github.com/appscode/stash/pkg/cmds/server"
	shell "github.com/codeskyblue/go-sh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	StashProjectRoot string
)

func (f *Framework) InstallStashOperator(kubeConfigPath string, options *srvr.ExtraOptions) {
	defer GinkgoRecover()
	sh := shell.NewSession()
	sh.SetEnv("APPSCODE_ENV", "dev")
	sh.SetEnv("STASH_IMAGE_TAG", options.StashImageTag)
	sh.SetDir(StashProjectRoot)

	args := []interface{}{"--namespace=" + f.namespace, "--docker-registry=" + options.DockerRegistry}

	runScript := filepath.Join("hack", "deploy", "stash.sh")

	By("Installing Stash")
	cmd := sh.Command(runScript, args...)
	err := cmd.Run()
	Expect(err).ShouldNot(HaveOccurred())

}

func (f *Framework) UninstallStashOperator() {
	sh := shell.NewSession()
	sh.SetDir(StashProjectRoot)

	args := []interface{}{"--uninstall", "--purge"}

	runScript := filepath.Join("hack", "deploy", "stash.sh")

	By("Uninstalling Stash")
	cmd := sh.Command(runScript, args...)
	err := cmd.Run()
	Expect(err).ShouldNot(HaveOccurred())
}
