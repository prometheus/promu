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
	dockerBuilderImageName = "prom/golang-builder"

	defaultGoVersion     = "1.5.4"
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
	// 1.6 "linux/mips64", "linux/mips64le",
	defaultMIPSPlatforms = []string{
		"",
	}
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
		if !viper.IsSet("crossbuild.platforms.mips") {
			viper.Set("crossbuild.platforms.mips", defaultMIPSPlatforms)
		}
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(crossbuildCmd)

	crossbuildCmd.Flags().String("go", "", "Golang builder version to use")
	crossbuildCmd.Flags().String("repo-path", "", "Project repository path")
	crossbuildCmd.Flags().String("main", "", "Main platforms to build")
	crossbuildCmd.Flags().String("arm", "", "ARM platforms to build")
	crossbuildCmd.Flags().String("powerpc", "", "PowerPC platforms to build")
	crossbuildCmd.Flags().String("mips", "", "MIPS platforms to build")

	viper.BindPFlag("go", crossbuildCmd.Flags().Lookup("go"))
	viper.BindPFlag("repository.path", crossbuildCmd.Flags().Lookup("repo-path"))
	viper.BindPFlag("crossbuild.platforms.main", crossbuildCmd.Flags().Lookup("main"))
	viper.BindPFlag("crossbuild.platforms.arm", crossbuildCmd.Flags().Lookup("arm"))
	viper.BindPFlag("crossbuild.platforms.powerpc", crossbuildCmd.Flags().Lookup("powerpc"))
	viper.BindPFlag("crossbuild.platforms.mips", crossbuildCmd.Flags().Lookup("mips"))

	viper.SetDefault("go", defaultGoVersion)
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
		goVersion        = viper.GetString("go")
		repoPath         = viper.GetString("repository.path")
		mainPlatforms    = viper.GetStringSlice("crossbuild.platforms.main")
		ARMPlatforms     = viper.GetStringSlice("crossbuild.platforms.arm")
		powerPCPlatforms = viper.GetStringSlice("crossbuild.platforms.powerpc")
		MIPSPlatforms    = viper.GetStringSlice("crossbuild.platforms.mips")

		dockerMainBuilderImage    = fmt.Sprintf("%s:%s-main", dockerBuilderImageName, goVersion)
		dockerARMBuilderImage     = fmt.Sprintf("%s:%s-arm", dockerBuilderImageName, goVersion)
		dockerPowerPCBuilderImage = fmt.Sprintf("%s:%s-powerpc", dockerBuilderImageName, goVersion)
		dockerMIPSBuilderImage    = fmt.Sprintf("%s:%s-mips", dockerBuilderImageName, goVersion)
	)

	if mainPlatformsParam := strings.Join(mainPlatforms[:], " "); mainPlatformsParam != "" {
		fmt.Println("> running the main builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerMainBuilderImage, "-i", repoPath, "-p", q(mainPlatformsParam))
	}

	if ARMPlatformsParam := strings.Join(ARMPlatforms[:], " "); ARMPlatformsParam != "" {
		fmt.Println("> running the ARM builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerARMBuilderImage, "-i", repoPath, "-p", q(ARMPlatformsParam))
	}

	if powerPCPlatformsParam := strings.Join(powerPCPlatforms[:], " "); powerPCPlatformsParam != "" {
		fmt.Println("> running the PowerPC builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerPowerPCBuilderImage, "-i", repoPath, "-p", q(powerPCPlatformsParam))
	}

	if MIPSPlatformsParam := strings.Join(MIPSPlatforms[:], " "); MIPSPlatformsParam != "" {
		fmt.Println("> running the MIPS builder docker image")
		sh("docker run --rm -t -v $PWD:/app",
			dockerMIPSBuilderImage, "-i", repoPath, "-p", q(MIPSPlatformsParam))
	}
}
