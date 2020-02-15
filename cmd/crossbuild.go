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
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	defaultMainPlatforms = []string{
		"linux/amd64", "linux/386",
		"darwin/amd64", "darwin/386",
		"windows/amd64", "windows/386",
		"freebsd/amd64", "freebsd/386",
		"openbsd/amd64", "openbsd/386",
		"netbsd/amd64", "netbsd/386",
		"dragonfly/amd64",
	}
	defaultARMPlatforms = []string{
		"linux/armv5", "linux/armv6", "linux/armv7", "linux/arm64",
		"freebsd/armv6", "freebsd/armv7",
		"openbsd/armv7",
		"netbsd/armv6", "netbsd/armv7",
	}
	defaultPowerPCPlatforms = []string{
		"aix/ppc64", "linux/ppc64", "linux/ppc64le",
	}
	defaultMIPSPlatforms = []string{
		"linux/mips", "linux/mipsle",
		"linux/mips64", "linux/mips64le",
	}
	defaultS390Platforms = []string{
		"linux/s390x",
	}
	armPlatformsAliases = map[string][]string{
		"linux/arm":   {"linux/armv5", "linux/armv6", "linux/armv7"},
		"freebsd/arm": {"freebsd/armv6", "freebsd/armv7"},
		"openbsd/arm": {"openbsd/armv7"},
		"netbsd/arm":  {"netbsd/armv6", "netbsd/armv7"},
	}
)

var (
	crossbuildcmd        = app.Command("crossbuild", "Crossbuild a Go project")
	crossBuildCgoFlagSet bool
	crossBuildCgoFlag    = crossbuildcmd.Flag("cgo", "Enable CGO using several docker images with different crossbuild toolchains.").
				PreAction(func(c *kingpin.ParseContext) error {
			crossBuildCgoFlagSet = true
			return nil
		}).Default("false").Bool()
	goFlagSet bool
	goFlag    = crossbuildcmd.Flag("go", "Golang builder version to use (e.g. 1.11)").
			PreAction(func(c *kingpin.ParseContext) error {
			goFlagSet = true
			return nil
		}).String()
	platformsFlagSet bool
	platformsFlag    = crossbuildcmd.Flag("platforms", "Space separated list of platforms to build").Short('p').
				PreAction(func(c *kingpin.ParseContext) error {
			platformsFlagSet = true
			return nil
		}).Strings()
	// kingpin doesn't currently support using the crossbuild command and the
	// crossbuild tarball subcommand at the same time, so we treat the
	// tarball subcommand as an optional arg
	tarballsSubcommand = crossbuildcmd.Arg("tarballs", "Optionally pass the string \"tarballs\" from cross-built binaries").String()
)

func runCrossbuild() {
	//Check required configuration
	if len(strings.TrimSpace(config.Repository.Path)) == 0 {
		log.Fatalf("missing required '%s' configuration", "repository.path")
	}
	if *tarballsSubcommand == "tarballs" {
		runCrossbuildTarballs()
		return
	}

	if crossBuildCgoFlagSet {
		config.Go.CGo = *crossBuildCgoFlag
	}
	if goFlagSet {
		config.Go.Version = *goFlag
	}
	if platformsFlagSet {
		config.Crossbuild.Platforms = *platformsFlag
	}

	var (
		unknownPlatforms []string
		platforms        = config.Crossbuild.Platforms
	)

	for _, platform := range platforms {
		switch {
		case stringInSlice(platform, defaultMainPlatforms):
		case stringInSlice(platform, defaultARMPlatforms):
		case stringInSlice(platform, defaultPowerPCPlatforms):
		case stringInSlice(platform, defaultMIPSPlatforms):
		case stringInSlice(platform, defaultS390Platforms):
		case stringInMapKeys(platform, armPlatformsAliases):
		default:
			unknownPlatforms = append(unknownPlatforms, platform)
		}
	}

	if len(unknownPlatforms) > 0 {
		warn(errors.Errorf("unknown/unhandled platforms: %s", unknownPlatforms))
	}

	sem := make(chan struct{}, *crossbuildJobs)
	errs := make([]error, 0, len(platforms))

	fmt.Printf("~ building up to %d concurrent crossbuilds\n", *crossbuildJobs)
	fmt.Printf("~ building up to %d concurrent binaries\n", *binaryJobs)

	// Launching builds concurrently
	for _, platform := range platforms {
		sem <- struct{}{}

		goos := platform[0:strings.Index(platform, "/")]
		goarch := platform[strings.Index(platform, "/")+1:]

		go func(goos string, goarch string) {
			fmt.Printf("< building platform %s/%s\n", goos, goarch)
			start := time.Now()
			runBuild(goos, goarch, "all")
			duration := time.Since(start)
			fmt.Printf("> %s/%s (built in %v)\n", goos, goarch, duration.Round(time.Millisecond))
			<-sem
		}(goos, goarch)
	}

	// Wait for builds to finish
	for {
		if len(sem) == 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			printErr(err)
		}

		fatal(errors.New("Crossbuild failed"))
	}
}
