package main

import (
	"os"

	"github.com/appscode/go/log"
	logs "github.com/appscode/go/log/golog"
	_ "github.com/appscode/stash/client/fake"
	_ "github.com/appscode/stash/client/internalclientset/scheme"
	_ "github.com/appscode/stash/client/scheme"
	"github.com/appscode/stash/pkg/cmds"
	_ "k8s.io/client-go/kubernetes/fake"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewCmdStash(Version).Execute(); err != nil {
		log.Infoln("Error in Stash Main:", err)
		os.Exit(1)
	}
	log.Infoln("Exiting Stash Main")
	os.Exit(0)
}
