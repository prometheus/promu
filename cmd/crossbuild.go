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
	"strings"

	"github.com/progrium/go-shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dockerBuilderImageName    = "prom/golang-builder"
	dockerMainBuilderImage    = fmt.Sprintf("%s:main", dockerBuilderImageName)
	dockerARMBuilderImage     = fmt.Sprintf("%s:arm", dockerBuilderImageName)
	dockerPowerPCBuilderImage = fmt.Sprintf("%s:powerpc", dockerBuilderImageName)
	//dockerMIPSBuilderImage       = fmt.Sprintf("%s:mips", dockerBuilderImageName)

	defaultMainPlatforms = []string{
		"linux/amd64", "linux/386", "darwin/amd64", "darwin/386", "windows/amd64", "windows/386",
		"freebsd/amd64", "freebsd/386", "openbsd/amd64", "openbsd/386", "netbsd/amd64", "netbsd/386",
		"dragonfly/amd64",
	}
	defaultARMPlatforms = []string{
		"linux/arm", "linux/arm64", "freebsd/arm", "openbsd/arm", "netbsd/arm",
	}
	defaultPowerPCPlatforms = []string{
		"linux/ppc64", "linux/ppc64le",
	}
	/*defaultMIPSPlatforms = []string{
		"linux/mips", "linux/mipsel",
	}*/
)

// crossbuildCmd represents the crossbuild command
var crossbuildCmd = &cobra.Command{
	Use:   "crossbuild",
	Short: "Crossbuild a Go project using Golang builder Docker images",
	Long:  `Crossbuild a Go project using Golang builder Docker images`,
	Run: func(cmd *cobra.Command, args []string) {
		runCrossbuild()
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet("crossbuild.platforms.main") {
			viper.Set("crossbuild.platforms.main", defaultMainPlatforms)
		}
		if !viper.IsSet("crossbuild.platforms.arm") {
			viper.Set("crossbuild.platforms.arm", defaultARMPlatforms)
		}
		if !viper.IsSet("crossbuild.platforms.powerpc") {
			viper.Set("crossbuild.platforms.powerpc", defaultPowerPCPlatforms)
		}
		//if !viper.IsSet("crossbuild.platforms.mips") {
		//	viper.SetDefault("crossbuild.platforms.mips", defaultMIPSPlatforms)
		//}
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(crossbuildCmd)

	crossbuildCmd.Flags().String("repo-path", "", "Project repository path")
	crossbuildCmd.Flags().String("main", "", "Main platforms to build")
	crossbuildCmd.Flags().String("arm", "", "ARM platforms to build")
	crossbuildCmd.Flags().String("powerpc", "", "PowerPC platforms to build")
	//crossbuildCmd.Flags().String("mips", "", "MIPS platforms to build")

	viper.BindPFlag("repository.path", crossbuildCmd.Flags().Lookup("repo-path"))
	viper.BindPFlag("crossbuild.platforms.main", crossbuildCmd.Flags().Lookup("main"))
	viper.BindPFlag("crossbuild.platforms.arm", crossbuildCmd.Flags().Lookup("arm"))
	viper.BindPFlag("crossbuild.platforms.powerpc", crossbuildCmd.Flags().Lookup("powerpc"))
	//viper.BindPFlag("crossbuild.platforms.mips", crossbuildCmd.Flags().Lookup("mips"))

	// Current bug in viper: SeDefault doesn't work with nested key
	//viper.SetDefault("crossbuild.platforms.main", defaultMainPlatforms)
	//viper.SetDefault("crossbuild.platforms.arm", defaultARMPlatforms)
	//viper.SetDefault("crossbuild.platforms.powerpc", defaultPowerPCPlatforms)
	//viper.SetDefault("crossbuild.platforms.mips", defaultMIPSPlatforms)
}

func runCrossbuild() {
	defer shell.ErrExit()
	shell.Tee = os.Stdout

	if viper.GetBool("verbose") {
		shell.Trace = true
	}

	var (
		repoPath         = viper.GetString("repository.path")
		mainPlatforms    = viper.GetStringSlice("crossbuild.platforms.main")
		ARMPlatforms     = viper.GetStringSlice("crossbuild.platforms.arm")
		powerPCPlatforms = viper.GetStringSlice("crossbuild.platforms.powerpc")
		//MIPSPlatforms    = viper.GetStringSlice("crossbuild.platforms.mips")
	)

	if len(mainPlatforms) > 0 {
		fmt.Println("> running the main builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerMainBuilderImage, "-i", repoPath, "-p", q(strings.Join(mainPlatforms[:], " ")))
	}

	if len(ARMPlatforms) > 0 {
		fmt.Println("> running the ARM builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerARMBuilderImage, "-i", repoPath, "-p", q(strings.Join(ARMPlatforms[:], " ")))
	}

	if len(powerPCPlatforms) > 0 {
		fmt.Println("> running the PowerPC builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerPowerPCBuilderImage, "-i", repoPath, "-p", q(strings.Join(powerPCPlatforms[:], " ")))
	}

	/*if len(MIPSPlatforms) > 0 {
		fmt.Println("> running the MIPS builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerMIPSBuilderImage, "-i", repoPath, "-p", q(strings.Join(MIPSPlatforms[:], " ")))
	}*/
}
