#!/bin/bash

# https://github.com/ellisonbg/antipackage
pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage

go get -u golang.org/x/tools/cmd/goimports
go get github.com/constabulary/gb/...
go install github.com/onsi/ginkgo/ginkgo
