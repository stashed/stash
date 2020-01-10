/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

	"stash.appscode.dev/stash/pkg/cmds"

	"github.com/appscode/go/runtime"
	"github.com/spf13/cobra/doc"
)

var (
	tplFrontMatter = template.Must(template.New("index").Parse(`---
title: Stash Operator
description: Stash Operator Reference
menu:
  docs_{{ "{{ .version }}" }}:
    identifier: operator
    name: Stash Operator
    parent: reference
    weight: 20
menu_name: docs_{{ "{{ .version }}" }}
---
`))

	_ = template.Must(tplFrontMatter.New("cmd").Parse(`---
title: {{ .Name }}
menu:
  docs_{{ "{{ .version }}" }}:
    identifier: {{ .ID }}
    name: {{ .Name }}
    parent: operator
{{- if .RootCmd }}
    weight: 0
{{ end }}
product_name: stash
section_menu_id: reference
menu_name: docs_{{ "{{ .version }}" }}
{{- if .RootCmd }}
url: /docs/{{ "{{ .version }}" }}/reference/operator/
aliases:
  - /docs/{{ "{{ .version }}" }}/reference/operator/operator/
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
			RootCmd bool
		}{
			strings.Replace(base, "_", "-", -1),
			name,
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
	err = doc.GenMarkdownTreeCustom(rootCmd, dir, filePrepender, linkHandler)
	if err != nil {
		log.Fatalln(err)
	}

	index := filepath.Join(dir, "_index.md")
	f, err := os.OpenFile(index, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	err = tplFrontMatter.ExecuteTemplate(f, "index", struct{}{})
	if err != nil {
		log.Fatalln(err)
	}
	if err := f.Close(); err != nil {
		log.Fatalln(err)
	}
}
