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

package logs

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gomodules.xyz/flags"
	"gomodules.xyz/wait"
	"k8s.io/klog/v2"
)

// ref:
// - https://github.com/kubernetes/component-base/blob/master/logs/logs.go

const logFlushFreqFlagName = "log-flush-frequency"

var logFlushFreq = pflag.Duration(logFlushFreqFlagName, 5*time.Second, "Maximum number of seconds between log flushes")

func init() {
	_ = flag.Set("stderrthreshold", "INFO")
}

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(logFlushFreqFlagName))
}

// KlogWriter serves as a bridge between the standard log package and the klog package.
type KlogWriter struct{}

// Write implements the io.Writer interface.
func (writer KlogWriter) Write(data []byte) (n int, err error) {
	klog.InfoDepth(1, string(data))
	return len(data), nil
}

// Init initializes logs the way we want for AppsCode codebase.
func Init(rootCmd *cobra.Command, printFlags bool) {
	klog.InitFlags(nil)
	pflag.CommandLine.SetNormalizeFunc(WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	InitLogs()

	if rootCmd == nil {
		// This branch only makes sense if Cobra is NOT used
		// If Cobra is used, set the rootCmd
		pflag.Parse()
		fs := pflag.CommandLine
		if printFlags {
			flags.PrintFlags(fs)
		}
		flags.LoggerOptions = flags.GetOptions(fs)
		return
	}

	fs := rootCmd.Flags()
	if fn := rootCmd.PersistentPreRunE; fn != nil {
		rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			if printFlags {
				flags.PrintFlags(fs)
			}
			flags.LoggerOptions = flags.GetOptions(fs)
			return fn(cmd, args)
		}
	} else if fn := rootCmd.PersistentPreRun; fn != nil {
		rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			if printFlags {
				flags.PrintFlags(fs)
			}
			flags.LoggerOptions = flags.GetOptions(fs)
			fn(cmd, args)
		}
	} else {
		rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			if printFlags {
				flags.PrintFlags(fs)
			}
			flags.LoggerOptions = flags.GetOptions(fs)
		}
	}
}

// WordSepNormalizeFunc changes all flags that contain "_" separators
func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.Replace(name, "_", "-", -1))
	}
	return pflag.NormalizedName(name)
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	log.SetOutput(KlogWriter{})
	log.SetFlags(0)
	// The default glog flush interval is 5 seconds.
	go wait.Forever(klog.Flush, *logFlushFreq)
}

// FlushLogs flushes logs immediately.
func FlushLogs() {
	klog.Flush()
}

// NewLogger creates a new log.Logger which sends logs to klog.Info.
func NewLogger(prefix string) *log.Logger {
	return log.New(KlogWriter{}, prefix, 0)
}

// GlogSetter is a setter to set glog level.
func GlogSetter(val string) (string, error) {
	var level klog.Level
	if err := level.Set(val); err != nil {
		return "", fmt.Errorf("failed set klog.logging.verbosity %s: %v", val, err)
	}
	return fmt.Sprintf("successfully set klog.logging.verbosity to %s", val), nil
}
