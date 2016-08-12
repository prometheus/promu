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

package cmd

import (
	"fmt"

	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
)

var (
	short bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and exit",
	Long:  `Print the version of promu, and various build and configuration information.`,
	Run: func(cmd *cobra.Command, args []string) {
		runVersion()
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(versionCmd)

	versionCmd.Flags().BoolVarP(&short, "short", "s", false, "Print shorter version")
}

func runVersion() {
	if short != false {
		fmt.Printf(version.Version)
		return
	}
	fmt.Println(version.Print("promu"))
	return
}
