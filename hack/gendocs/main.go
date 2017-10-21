package main

import (
	"fmt"
	"github.com/appscode/go/runtime"
	"github.com/appscode/stash/pkg/cmds"
	"github.com/spf13/cobra/doc"
	"log"
	"os"
)

// ref: https://github.com/spf13/cobra/blob/master/doc/md_docs.md
func main() {
	rootCmd := cmds.NewCmdStash("")
	dir := runtime.GOPath() + "/src/github.com/appscode/stash/docs/reference"
	fmt.Printf("Generating cli markdown tree in: %v\n", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	doc.GenMarkdownTree(rootCmd, dir)
}
