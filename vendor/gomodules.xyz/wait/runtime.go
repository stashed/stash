/*
Copyright 2014 The Kubernetes Authors.
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

package wait

var (
	// reallyCrash controls the behavior of handleCrash and now defaults
	// true. It's still exposed so components can optionally set to false
	// to restore prior behavior.
	reallyCrash = true
)

// panicHandlers is a list of functions which will be invoked when a panic happens.
var panicHandlers = []func(interface{}){}

// handleCrash simply catches a crash and actually crashes.
func handleCrash() {
	if r := recover(); r != nil {
		for _, fn := range panicHandlers {
			fn(r)
		}
		if reallyCrash {
			// Actually proceed to panic.
			panic(r)
		}
	}
}
