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

	"github.com/mcuadros/go-version"
	"github.com/progrium/go-shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dockerBuilderImageName = "quay.io/prometheus/golang-builder"

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
	defaultMIPSPlatforms = []string{
		"linux/mips64", "linux/mips64le",
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
		if err := hasRequiredConfigurations("repository.path"); err != nil {
			fatal(err)
		}
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(crossbuildCmd)

	crossbuildCmd.Flags().String("go", "", "Golang builder version to use")
	crossbuildCmd.Flags().StringP("platforms", "p", "", "Platforms to build")

	viper.BindPFlag("go", crossbuildCmd.Flags().Lookup("go"))
	viper.BindPFlag("crossbuild.platforms", crossbuildCmd.Flags().Lookup("platforms"))

	viper.SetDefault("go", defaultGoVersion)
	// Current bug in viper: SeDefault doesn't work with nested key
	// platforms := defaultMainPlatforms
	// platforms = append(platforms, defaultARMPlatforms...)
	// platforms = append(platforms, defaultPowerPCPlatforms...)
	// platforms = append(platforms, defaultMIPSPlatforms...)
	// viper.SetDefault("crossbuild.platforms", platforms)
}

func runCrossbuild() {
	defer shell.ErrExit()
	shell.Tee = os.Stdout

	if viper.GetBool("verbose") {
		shell.Trace = true
	}

	var (
		mainPlatforms    []string
		armPlatforms     []string
		powerPCPlatforms []string
		mipsPlatforms    []string
		unknownPlatforms []string

		goVersion = viper.GetString("go")
		repoPath  = viper.GetString("repository.path")
		platforms = viper.GetStringSlice("crossbuild.platforms")

		dockerMainBuilderImage    = fmt.Sprintf("%s:%s-main", dockerBuilderImageName, goVersion)
		dockerARMBuilderImage     = fmt.Sprintf("%s:%s-arm", dockerBuilderImageName, goVersion)
		dockerPowerPCBuilderImage = fmt.Sprintf("%s:%s-powerpc", dockerBuilderImageName, goVersion)
		dockerMIPSBuilderImage    = fmt.Sprintf("%s:%s-mips", dockerBuilderImageName, goVersion)
	)

	for _, platform := range platforms {
		switch {
		case stringInSlice(platform, defaultMainPlatforms):
			mainPlatforms = append(mainPlatforms, platform)
		case stringInSlice(platform, defaultARMPlatforms):
			armPlatforms = append(armPlatforms, platform)
		case stringInSlice(platform, defaultPowerPCPlatforms):
			powerPCPlatforms = append(powerPCPlatforms, platform)
		case stringInSlice(platform, defaultMIPSPlatforms):
			mipsPlatforms = append(mipsPlatforms, platform)
		default:
			unknownPlatforms = append(unknownPlatforms, platform)
		}
	}

	if len(unknownPlatforms) > 0 {
		warn(fmt.Errorf("unknown/unhandled platforms: %s", unknownPlatforms))
	}

	if mainPlatformsParam := strings.Join(mainPlatforms[:], " "); mainPlatformsParam != "" {
		fmt.Println("> running the main builder docker image")
		if err := docker("run --rm -t -v $PWD:/app", dockerMainBuilderImage, "-i", repoPath, "-p", q(mainPlatformsParam)); err != nil {
			fatalMsg("The main builder docker image exited unexpectedly", err)
		}
	}

	if armPlatformsParam := strings.Join(armPlatforms[:], " "); armPlatformsParam != "" {
		fmt.Println("> running the ARM builder docker image")
		if err := docker("run --rm -t -v $PWD:/app", dockerARMBuilderImage, "-i", repoPath, "-p", q(armPlatformsParam)); err != nil {
			fatalMsg("The ARM builder docker image exited unexpectedly", err)
		}
	}

	if powerPCPlatformsParam := strings.Join(powerPCPlatforms[:], " "); powerPCPlatformsParam != "" {
		fmt.Println("> running the PowerPC builder docker image")
		if err := docker("run --rm -t -v $PWD:/app", dockerPowerPCBuilderImage, "-i", repoPath, "-p", q(powerPCPlatformsParam)); err != nil {
			fatalMsg("The PowerPC builder docker image exited unexpectedly", err)
		}
	}

	if mipsPlatformsParam := strings.Join(mipsPlatforms[:], " "); mipsPlatformsParam != "" {
		if version.Compare(goVersion, "1.6", ">=") {
			fmt.Println("> running the MIPS builder docker image")
			if err := docker("run --rm -t -v $PWD:/app", dockerMIPSBuilderImage, "-i", repoPath, "-p", q(mipsPlatformsParam)); err != nil {
				fatalMsg("The MIPS builder docker image exited unexpectedly", err)
			}
		} else {
			warn(fmt.Errorf("MIPS architectures are only available with Go 1.6+"))
		}
	}
}
