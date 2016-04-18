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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// crossbuildTarballsCmd represents the crossbuild tarballs command
var crossbuildTarballsCmd = &cobra.Command{
	Use:   "tarballs",
	Short: "Create tarballs from cross-built binaries",
	Long:  `Create tarballs from cross-built binaries`,
	Run: func(cmd *cobra.Command, args []string) {
		runCrossbuildTarballs()
	},
}

// init prepares cobra flags
func init() {
	crossbuildCmd.AddCommand(crossbuildTarballsCmd)
}

func runCrossbuildTarballs() {

	dirs, err := ioutil.ReadDir(".build")
	if err != nil {
		fatal(err)
	}

	fmt.Println(">> building release tarballs")
	for _, dir := range dirs {
		viper.Set("tarball.prefix", ".tarballs")

		if platform := strings.Split(dir.Name(), "-"); len(platform) == 2 {
			os.Setenv("GOOS", platform[0])
			os.Setenv("GOARCH", platform[1])
		} else {
			if err := fmt.Errorf("bad .build/%s directory naming, should be <GOOS>-<GOARCH>", platform); err != nil {
				fatal(err)
			}
		}

		runTarball(filepath.Join(".build", dir.Name()))
	}

	defer os.Unsetenv("GOOS")
	defer os.Unsetenv("GOARCH")
}
