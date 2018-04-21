package main

import (
	"log"

	"github.com/appscode/osm/cmds"
)

func main() {
	if err := cmds.NewCmdOsm().Execute(); err != nil {
		log.Fatal(err)
	}
}
