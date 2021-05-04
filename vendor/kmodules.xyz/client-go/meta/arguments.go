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

// ref: https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/util/arguments.go

package meta

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

func UpsertArgumentList(baseArgs []string, overrideArgs []string, protectedFlags ...string) []string {
	out := make([]string, 0, len(baseArgs)+len(overrideArgs))

	baseMap := make(map[string]int, len(baseArgs))
	for i, arg := range baseArgs {
		out = append(out, arg) // copy to out

		if !strings.HasPrefix(arg, "--") {
			continue
		}
		idx := strings.IndexRune(arg, '=')
		if idx < 0 {
			continue
		}
		baseMap[arg[:idx]] = i
	}

	protectedSet := make(map[string]string, len(protectedFlags))
	for _, flag := range protectedFlags {
		protectedSet[flag] = ""
	}

	var idx int
	for _, arg := range overrideArgs {
		if !strings.HasPrefix(arg, "--") {
			goto Append
		}
		idx = strings.IndexRune(arg, '=')
		if idx < 0 {
			goto Append
		}

		if _, found := protectedSet[arg[:idx]]; found {
			continue
		}

		if idx, found := baseMap[arg[:idx]]; found {
			out[idx] = arg // overwrite
			continue
		}

	Append:
		out = append(out, arg) // append to out
	}
	return out
}

// BuildArgumentListFromMap takes two string-string maps, one with the base arguments and one with optional override arguments
func BuildArgumentListFromMap(baseArguments map[string]string, overrideArguments map[string]string) []string {
	var command []string
	for k, v := range overrideArguments {
		// values of "" are allowed as well
		command = append(command, fmt.Sprintf("--%s=%s", k, v))
	}
	for k, v := range baseArguments {
		if _, overrideExists := overrideArguments[k]; !overrideExists {
			command = append(command, fmt.Sprintf("--%s=%s", k, v))
		}
	}
	return command
}

// ParseArgumentListToMap parses a CLI argument list in the form "--foo=bar" to a string-string map
func ParseArgumentListToMap(arguments []string) map[string]string {
	resultingMap := map[string]string{}
	for i, arg := range arguments {
		key, val, err := parseArgument(arg)

		// Ignore if the first argument doesn't satisfy the criteria, it's most often the binary name
		// Warn in all other cases, but don't error out. This can happen only if the user has edited the argument list by hand, so they might know what they are doing
		if err != nil {
			if i != 0 {
				klog.Warningf("WARNING: The component argument %q could not be parsed correctly. The argument must be of the form %q. Skipping...", arg, "--")
			}
			continue
		}

		resultingMap[key] = val
	}
	return resultingMap
}

// ReplaceArgument gets a command list; converts it to a map for easier modification, runs the provided function that
// returns a new modified map, and then converts the map back to a command string slice
func ReplaceArgument(command []string, argMutateFunc func(map[string]string) map[string]string) []string {
	argMap := ParseArgumentListToMap(command)

	// Save the first command (the executable) if we're sure it's not an argument (i.e. no --)
	var newCommand []string
	if len(command) > 0 && !strings.HasPrefix(command[0], "--") {
		newCommand = append(newCommand, command[0])
	}
	newArgMap := argMutateFunc(argMap)
	newCommand = append(newCommand, BuildArgumentListFromMap(newArgMap, map[string]string{})...)
	return newCommand
}

// parseArgument parses the argument "--foo=bar" to "foo" and "bar"
func parseArgument(arg string) (string, string, error) {
	if !strings.HasPrefix(arg, "--") {
		return "", "", fmt.Errorf("the argument should start with '--'")
	}
	if !strings.Contains(arg, "=") {
		return "", "", fmt.Errorf("the argument should have a '=' between the flag and the value")
	}
	// Remove the starting --
	arg = strings.TrimPrefix(arg, "--")
	// Split the string on =. Return only two substrings, since we want only key/value, but the value can include '=' as well
	keyvalSlice := strings.SplitN(arg, "=", 2)

	// Make sure both a key and value is present
	if len(keyvalSlice) != 2 {
		return "", "", fmt.Errorf("the argument must have both a key and a value")
	}
	if len(keyvalSlice[0]) == 0 {
		return "", "", fmt.Errorf("the argument must have a key")
	}

	return keyvalSlice[0], keyvalSlice[1], nil
}
