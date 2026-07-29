// Copyright © 2016 Prometheus Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

// Verbose enables verbose output
var Verbose bool

// RunCommand executes a shell command.
func RunCommand(name string, arg ...string) error {
	if Verbose {
		cmdText := name + " " + strings.Join(arg, " ")
		fmt.Fprintln(os.Stderr, " + ", cmdText)
	}
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SplitParameters splits shell command parameters, taking quoting into account.
// Single and double quotes group text without becoming part of the parameter.
func SplitParameters(s string) []string {
	var (
		params  []string
		current strings.Builder
		inParam bool
		quote   rune
	)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
			inParam = true
		case unicode.IsSpace(r):
			if inParam {
				params = append(params, current.String())
				current.Reset()
				inParam = false
			}
		default:
			current.WriteRune(r)
			inParam = true
		}
	}
	if inParam {
		params = append(params, current.String())
	}
	return params
}
