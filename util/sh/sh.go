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
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Verbose enables verbose output
var Verbose bool

// RunCommand executes a shell command.
func RunCommand(name string, arg ...string) error {
	var cmdText string

	if Verbose {
		cmdText = name + " " + strings.Join(QuoteParams(arg), " ")
		fmt.Fprintln(os.Stderr, " + ", cmdText)
	}

	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	return errors.Wrap(err, "command failed: "+cmdText)
}

// RunCommandWithEnv executes a shell command.
func RunCommandWithEnv(name string, dir string, env []string, arg ...string) error {
	var cmdText string

	if Verbose {
		if len(dir) > 0 {
			cmdText = "PWD=" + dir + " "
		}
		cmdText = cmdText + strings.Join(env, " ") + " " + name + " " + strings.Join(QuoteParams(arg), " ")
		fmt.Fprintln(os.Stderr, " +", cmdText)
	}

	cmd := exec.Command(name, arg...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if len(dir) > 0 {
		cmd.Dir = dir
	}

	err := cmd.Run()

	return errors.Wrap(err, "command failed: "+cmdText)
}

// SplitParameters splits shell command parameters, taking quoting in account.
func SplitParameters(s string) []string {
	r := regexp.MustCompile(`'[^']*'|[^ ]+`)
	return r.FindAllString(s, -1)
}

// QuoteParams returns params array ready for display
func QuoteParams(params []string) []string {
	quoted := make([]string, len(params))

	for k, v := range params {
		if strings.Index(v, " ") != -1 && string(v[0]) != `'` && string(v[0]) != `"` {
			quoted[k] = `"` + strings.ReplaceAll(v, `"`, `\\"`) + `"`
		} else {
			quoted[k] = v
		}
	}

	return quoted
}
