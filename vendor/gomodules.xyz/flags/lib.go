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
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gomodules.xyz/sets"
)

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

func PrintFlags(fs *pflag.FlagSet, list ...string) {
	bl := sets.NewString("secret", "token", "password", "credential")
	if len(list) > 0 {
		bl.Insert(list...)
	}
	fs.VisitAll(func(flag *pflag.Flag) {
		name := strings.ToLower(flag.Name)
		val := flag.Value.String()
		for _, keyword := range bl.UnsortedList() {
			if strings.Contains(name, keyword) {
				if val != "" {
					val = "***REDACTED***"
				}
				break
			}
		}
		log.Printf("FLAG: --%s=%q", flag.Name, val)
	})
}
