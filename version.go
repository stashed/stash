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
