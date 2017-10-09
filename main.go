package main

import (
	"os"

	logs "github.com/appscode/go/log/golog"
	_ "github.com/appscode/stash/client/fake"
	_ "github.com/appscode/stash/client/internalclientset/scheme"
	"github.com/appscode/stash/pkg/cmds"
	_ "github.com/appscode/stash/client/scheme"
	_ "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/pkg/api/v1"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewCmdStash(Version).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
