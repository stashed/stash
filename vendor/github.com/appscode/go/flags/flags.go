package flags

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	flag.Set("logtostderr", "true")
}

// Init all the pflags and all underlying go flags
// All go flags of the underlying library converted to pflag and can set
// from terminal as flags.
func InitFlags() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

// Sets log level in runtime.
func SetLogLevel(l int) {
	var mu sync.Mutex
	mu.Lock()
	flag.Set("v", strconv.Itoa(l))
	mu.Unlock()
}

// Checks if a flag value in a command has been provided by the user
// Or not. The ordering of the flags can be set for nested flags.
func EnsureRequiredFlags(cmd *cobra.Command, name ...string) {
	for _, n := range name {
		flag := cmd.Flag(n)
		if flag == nil {
			// term.Fatalln(fmt.Printf("flag [--%v] go flag defined but called required.", flag.Name))
			continue
		}
		if !flag.Changed {
			fmt.Printf("flag [%v] is required but not provided.", flag.Name)
			os.Exit(3) // exit code 3 required for icinga plugins to indicate UNKNOWN state
		}
	}
}

// Checks for alternetable flags. One of the provided flags
// must needs to be set.
func EnsureAlterableFlags(cmd *cobra.Command, name ...string) {
	provided := false
	flagNames := ""
	for i, n := range name {
		flag := cmd.Flag(n)
		if i >= 1 {
			flagNames = flagNames + "/"
		}
		flagNames = flagNames + "--" + flag.Name
		if flag.Changed == true {
			provided = true
			break
		}
	}
	if provided == false {
		fmt.Printf("One of flag [ %v ] must needs to be set.", flagNames)
		os.Exit(3) // exit code 3 required for icinga plugins to indicate UNKNOWN state
	}
}

func DumpAll() {
	pflag.VisitAll(func(flag *pflag.Flag) {
		log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
