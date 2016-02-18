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
	"os"
	"runtime"

	"github.com/progrium/go-shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	tmpDir = ".release"
)

// tarballCmd represents the tarball command
var tarballCmd = &cobra.Command{
	Use:   "tarball [<prefix>] [<binaries-location>]",
	Short: "Create a tarball from the builded Go project",
	Run: func(cmd *cobra.Command, args []string) {
		prefix := optArg(args, 0, ".")
		binariesLocation := optArg(args, 1, ".")
		runTarball(prefix, binariesLocation)
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(tarballCmd)
}

func runTarball(prefix string, binariesLocation string) {
	defer shell.ErrExit()

	if viper.GetBool("verbose") {
		shell.Trace = true
		shell.Tee = os.Stdout
	}

	info := NewProjectInfo()

	var (
		binaries []Binary
		ext      string
	)

	os.Mkdir(tmpDir, 0777)
	defer sh("rm -rf", tmpDir)

	projectFiles := viper.GetStringSlice("tarball.files")
	for _, file := range projectFiles {
		sh("cp -a", file, tmpDir)
	}

	err := viper.UnmarshalKey("build.binaries", &binaries)
	fatalMsg(err, "Failed to Unmashal binaries :")

	for _, binary := range binaries {
		binaryName := fmt.Sprintf("%s%s", binary.Name, ext)
		sh("cp -a", shell.Path(binariesLocation, binaryName), tmpDir)
	}

	tar := fmt.Sprintf("%s-%s.%s-%s.tar.gz", info.Name, info.Version, runtime.GOOS, runtime.GOARCH)
	sh("tar zcf", shell.Path(prefix, tar), "-C", tmpDir, ".")
}
