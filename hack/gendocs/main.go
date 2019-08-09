package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/appscode/go/runtime"
	"github.com/spf13/cobra/doc"
	"stash.appscode.dev/stash/pkg/cmds"
)

const (
	version = "0.9.0-rc.0"
)

var (
	tplFrontMatter = template.Must(template.New("index").Parse(`---
title: Stash Operator
description: Stash Operator Reference
menu:
  product_stash_{{ .Version }}:
    identifier: operator
    name: Stash Operator
    parent: reference
    weight: 20
menu_name: product_stash_{{ .Version }}
---
`))

	_ = template.Must(tplFrontMatter.New("cmd").Parse(`---
title: {{ .Name }}
menu:
  product_stash_{{ .Version }}:
    identifier: {{ .ID }}
    name: {{ .Name }}
    parent: operator
{{- if .RootCmd }}
    weight: 0
{{ end }}
product_name: stash
section_menu_id: reference
menu_name: product_stash_{{ .Version }}
{{- if .RootCmd }}
url: /products/stash/{{ .Version }}/reference/operator/
aliases:
  - /products/stash/{{ .Version }}/reference/operator/operator/
{{ end }}
---
`))
)

// ref: https://github.com/spf13/cobra/blob/master/doc/md_docs.md
func main() {
	rootCmd := cmds.NewRootCmd()
	dir := runtime.GOPath() + "/src/stash.appscode.dev/docs/docs/reference/operator"
	fmt.Printf("Generating cli markdown tree in: %v\n", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatalln(err)
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatalln(err)
	}

	filePrepender := func(filename string) string {
		filename = filepath.Base(filename)
		base := strings.TrimSuffix(filename, path.Ext(filename))
		name := strings.Title(strings.Replace(base, "_", " ", -1))
		parts := strings.Split(name, " ")
		if len(parts) > 1 {
			name = strings.Join(parts[1:], " ")
		}
		data := struct {
			ID      string
			Name    string
			Version string
			RootCmd bool
		}{
			strings.Replace(base, "_", "-", -1),
			name,
			version,
			!strings.ContainsRune(base, '_'),
		}
		var buf bytes.Buffer
		if err := tplFrontMatter.ExecuteTemplate(&buf, "cmd", data); err != nil {
			log.Fatalln(err)
		}
		return buf.String()
	}

	linkHandler := func(name string) string {
		return "/docs/reference/operator/" + name
	}
	doc.GenMarkdownTreeCustom(rootCmd, dir, filePrepender, linkHandler)

	index := filepath.Join(dir, "_index.md")
	f, err := os.OpenFile(index, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	err = tplFrontMatter.ExecuteTemplate(f, "index", struct{ Version string }{version})
	if err != nil {
		log.Fatalln(err)
	}
	if err := f.Close(); err != nil {
		log.Fatalln(err)
	}
}
