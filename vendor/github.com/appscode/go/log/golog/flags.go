package golog

import (
	"fmt"

	"github.com/spf13/pflag"
)

type Options struct {
	ToStderr        bool   // The -logtostderr flag.
	AlsoToStderr    bool   // The -alsologtostderr flag.
	StderrThreshold string // The -stderrthreshold flag.
	Verbosity       string // V logging level, the value of the -v flag/
}

func ParseFlags(fs *pflag.FlagSet) Options {
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
	return []string{
		fmt.Sprintf("--logtostderr=%v", opt.ToStderr),
		fmt.Sprintf("--alsologtostderr=%v", opt.AlsoToStderr),
		fmt.Sprintf("--v=%v", opt.Verbosity),
		fmt.Sprintf("--stderrthreshold=%v", opt.StderrThreshold),
	}
}
