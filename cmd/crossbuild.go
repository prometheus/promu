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
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/pkg/errors"
	"go.uber.org/atomic"

	"github.com/prometheus/promu/util/sh"
)

var (
	dockerBuilderImageName = "quay.io/prometheus/golang-builder"

	defaultPlatforms = []string{
		"aix/ppc64",
		"darwin/amd64",
		"darwin/arm64",
		"dragonfly/amd64",
		"freebsd/386",
		"freebsd/amd64",
		"freebsd/arm64",
		"freebsd/armv6",
		"freebsd/armv7",
		"illumos/amd64",
		"linux/386",
		"linux/amd64",
		"linux/arm64",
		"linux/armv5",
		"linux/armv6",
		"linux/armv7",
		"linux/mips",
		"linux/mips64",
		"linux/mips64le",
		"linux/mipsle",
		"linux/ppc64",
		"linux/ppc64le",
		"linux/riscv64",
		"linux/s390x",
		"netbsd/386",
		"netbsd/amd64",
		"netbsd/arm64",
		"netbsd/armv6",
		"netbsd/armv7",
		"openbsd/386",
		"openbsd/amd64",
		"openbsd/arm64",
		"openbsd/armv7",
		"windows/386",
		"windows/amd64",
		"windows/arm64",
	}
)

var (
	crossbuildcmd        = app.Command("crossbuild", "Crossbuild a Go project using Golang builder Docker images")
	crossBuildCgoFlagSet bool
	crossBuildCgoFlag    = crossbuildcmd.Flag("cgo", "Enable CGO using several docker images with different crossbuild toolchains.").
				PreAction(func(c *kingpin.ParseContext) error {
			crossBuildCgoFlagSet = true
			return nil
		}).Default("false").Bool()
	parallelFlag       = crossbuildcmd.Flag("parallelism", "How many builds to run in parallel").Default("1").Int()
	parallelThreadFlag = crossbuildcmd.Flag("parallelism-thread", "Index of the parallel build").Default("-1").Int()
	goFlagSet          bool
	goFlag             = crossbuildcmd.Flag("go", "Golang builder version to use (e.g. 1.11)").
				PreAction(func(c *kingpin.ParseContext) error {
			goFlagSet = true
			return nil
		}).String()
	platformsFlagSet bool
	platformsFlag    = crossbuildcmd.Flag("platforms", "Regexp match platforms to build, may be used multiple times.").Short('p').
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
		allPlatforms     []string
		unknownPlatforms []string

		cgo       = config.Go.CGo
		goVersion = config.Go.Version
		repoPath  = config.Repository.Path
		platforms = config.Crossbuild.Platforms

		dockerBaseBuilderImage = fmt.Sprintf("%s:%s-base", dockerBuilderImageName, goVersion)
		dockerMainBuilderImage = fmt.Sprintf("%s:%s-main", dockerBuilderImageName, goVersion)
	)

	var filteredPlatforms []string
	for _, platform := range platforms {
		p := regexp.MustCompile(platform)
		if filteredPlatforms = inSliceRE(p, defaultPlatforms); len(filteredPlatforms) > 0 {
			allPlatforms = append(allPlatforms, filteredPlatforms...)
		} else {
			unknownPlatforms = append(unknownPlatforms, platform)
		}
	}

	// Remove duplicates, e.g. if linux/arm and linux/arm64 is specified, there
	// would be linux/arm64 twice in the platforms without this.
	allPlatforms = removeDuplicates(allPlatforms)

	if len(unknownPlatforms) > 0 {
		warn(errors.Errorf("unknown/unhandled platforms: %s", unknownPlatforms))
	}

	if !cgo {
		// In non-CGO, use the `base` image without any crossbuild toolchain.
		pg := &platformGroup{"base", dockerBaseBuilderImage, allPlatforms}
		if err := pg.Build(repoPath); err != nil {
			fatal(errors.Wrapf(err, "The %s builder docker image exited unexpectedly", pg.Name))
		}
	} else {
		// In CGO, use the `main` image with crossbuild toolchain.
		pg := &platformGroup{"main", dockerMainBuilderImage, allPlatforms}
		if err := pg.Build(repoPath); err != nil {
			fatal(errors.Wrapf(err, "The %s builder docker image exited unexpectedly", pg.Name))
		}
	}
}

type platformGroup struct {
	Name        string
	DockerImage string
	Platforms   []string
}

func (pg platformGroup) Build(repoPath string) error {
	if *parallelThreadFlag != -1 {
		return pg.buildThread(repoPath, *parallelThreadFlag)
	}
	err := sh.RunCommand("docker", "pull", pg.DockerImage)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(*parallelFlag)
	atomicErr := atomic.NewError(nil)
	for p := 0; p < *parallelFlag; p++ {
		go func(p int) {
			defer wg.Done()
			if err := pg.buildThread(repoPath, p); err != nil {
				atomicErr.Store(err)
			}
		}(p)
	}
	wg.Wait()
	return atomicErr.Load()
}

func (pg platformGroup) buildThread(repoPath string, p int) error {
	minb := p * len(pg.Platforms) / *parallelFlag
	maxb := (p + 1) * len(pg.Platforms) / *parallelFlag
	if maxb > len(pg.Platforms) {
		maxb = len(pg.Platforms)
	}
	platformsParam := strings.Join(pg.Platforms[minb:maxb], " ")
	if len(platformsParam) == 0 {
		return nil
	}

	fmt.Printf("> running the %s builder docker image\n", pg.Name)

	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "couldn't get current working directory")
	}

	ctrName := "promu-crossbuild-" + pg.Name + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.Itoa(p)
	err = sh.RunCommand("docker", "create", "-t",
		"--name", ctrName,
		pg.DockerImage,
		"-i", repoPath,
		"-p", platformsParam)
	if err != nil {
		return err
	}

	err = sh.RunCommand("docker", "cp",
		cwd+"/.",
		ctrName+":/app/")
	if err != nil {
		return err
	}

	err = sh.RunCommand("docker", "start", "-a", ctrName)
	if err != nil {
		return err
	}

	err = sh.RunCommand("docker", "cp", "-a",
		ctrName+":/app/.build/.",
		cwd+"/.build")
	if err != nil {
		return err
	}
	return sh.RunCommand("docker", "rm", "-f", ctrName)
}

func removeDuplicates(strings []string) []string {
	keys := map[string]struct{}{}
	list := []string{}
	for _, s := range strings {
		if _, ok := keys[s]; !ok {
			list = append(list, s)
			keys[s] = struct{}{}
		}
	}
	sort.Strings(list)
	return list
}
