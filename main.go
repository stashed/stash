package main

import (
	"os"

	logs "github.com/appscode/go/log/golog"
	_ "github.com/appscode/stash/client/fake"
	_ "github.com/appscode/stash/client/internalclientset/scheme"
	_ "github.com/appscode/stash/client/scheme"
	"github.com/appscode/stash/pkg/cmds"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/client-go/kubernetes/fake"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewCmdStash(Version).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
