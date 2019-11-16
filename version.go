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
	v "github.com/appscode/go/version"
)

var (
	Version         string
	VersionStrategy string
	GitTag          string
	GitBranch       string
	CommitHash      string
	CommitTimestamp string
	GoVersion       string
	Compiler        string
	Platform        string
)

func init() {
	v.Version.Version = Version
	v.Version.VersionStrategy = VersionStrategy
	v.Version.GitTag = GitTag
	v.Version.GitBranch = GitBranch
	v.Version.CommitHash = CommitHash
	v.Version.CommitTimestamp = CommitTimestamp
	v.Version.GoVersion = GoVersion
	v.Version.Compiler = Compiler
	v.Version.Platform = Platform
}
