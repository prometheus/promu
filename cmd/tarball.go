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

	"github.com/progrium/go-shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// tarballCmd represents the tarball command
var tarballCmd = &cobra.Command{
	Use:   "tarball [<binaries-location>]",
	Short: "Create a tarball from the builded Go project",
	Long:  `Create a tarball from the builded Go project`,
	Run: func(cmd *cobra.Command, args []string) {
		binariesLocation := optArg(args, 0, ".")
		runTarball(binariesLocation)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet("tarball.prefix") {
			viper.Set("tarball.prefix", ".")
		}
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(tarballCmd)

	tarballCmd.Flags().String("prefix", "", "Specific dir to store tarballs (default is .)")

	viper.BindPFlag("tarball.prefix", tarballCmd.Flags().Lookup("prefix"))
}

func runTarball(binariesLocation string) {
	defer shell.ErrExit()
	shell.Tee = os.Stdout

	if viper.GetBool("verbose") {
		shell.Trace = true
	}

	info := NewProjectInfo()

	var (
		prefix = viper.GetString("tarball.prefix")
		tmpDir = ".release"

		binaries []Binary
		ext      string
	)

	if goos == "windows" {
		ext = ".exe"
	}

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

	if !fileExists(prefix) {
		os.Mkdir(prefix, 0777)
	}

	tar := fmt.Sprintf("%s-%s.%s-%s.tar.gz", info.Name, info.Version, goos, goarch)
	fmt.Println(" >  ", tar)
	sh("tar zcf", shell.Path(prefix, tar), "-C", tmpDir, ".")
}
