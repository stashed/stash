/*
Copyright AppsCode Inc. and Contributors

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

package flags

import (
	"fmt"

	"github.com/spf13/pflag"
)

var (
	LoggerOptions Options
)

type Options struct {
	ToStderr        bool   // The -logtostderr flag.
	AlsoToStderr    bool   // The -alsologtostderr flag.
	StderrThreshold string // The -stderrthreshold flag.
	Verbosity       string // V logging level, the value of the -v flag/
}

func GetOptions(fs *pflag.FlagSet) Options {
	var opt Options

	opt.ToStderr, _ = fs.GetBool("logtostderr")
	opt.AlsoToStderr, _ = fs.GetBool("alsologtostderr")
	if f := fs.Lookup("v"); f != nil {
		opt.Verbosity = f.Value.String()
	}
	if f := fs.Lookup("stderrthreshold"); f != nil {
		opt.StderrThreshold = f.Value.String()
	}

	return opt
}

func (opt Options) ToFlags() []string {
	fs := []string{
		fmt.Sprintf("--logtostderr=%v", opt.ToStderr),
		fmt.Sprintf("--alsologtostderr=%v", opt.AlsoToStderr),
	}
	if opt.Verbosity != "" {
		fs = append(fs, fmt.Sprintf("--v=%v", opt.Verbosity))
	}
	if opt.StderrThreshold != "" {
		fs = append(fs, fmt.Sprintf("--stderrthreshold=%v", opt.StderrThreshold))
	}
	return fs
}
