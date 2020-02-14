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
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/promu/util/sh"
)

var (
	dockerBuilderImageName = "quay.io/prometheus/golang-builder"

	defaultMainPlatforms = []string{
		"linux/amd64", "linux/386", "darwin/amd64", "darwin/386", "windows/amd64", "windows/386",
		"freebsd/amd64", "freebsd/386", "openbsd/amd64", "openbsd/386", "netbsd/amd64", "netbsd/386",
		"dragonfly/amd64",
	}
	defaultARMPlatforms = []string{
		"linux/armv5", "linux/armv6", "linux/armv7", "linux/arm64", "freebsd/armv6", "freebsd/armv7",
		"openbsd/armv7", "netbsd/armv6", "netbsd/armv7",
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
	crossbuildcmd        = app.Command("crossbuild", "Crossbuild a Go project using Golang builder Docker images")
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

	dockerCopyMutex sync.Mutex
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
		mainPlatforms    []string
		armPlatforms     []string
		powerPCPlatforms []string
		mipsPlatforms    []string
		s390xPlatforms   []string
		unknownPlatforms []string

		cgo       = config.Go.CGo
		goVersion = config.Go.Version
		repoPath  = config.Repository.Path
		platforms = config.Crossbuild.Platforms

		dockerBaseBuilderImage    = fmt.Sprintf("%s:%s-base", dockerBuilderImageName, goVersion)
		dockerMainBuilderImage    = fmt.Sprintf("%s:%s-main", dockerBuilderImageName, goVersion)
		dockerARMBuilderImage     = fmt.Sprintf("%s:%s-arm", dockerBuilderImageName, goVersion)
		dockerPowerPCBuilderImage = fmt.Sprintf("%s:%s-powerpc", dockerBuilderImageName, goVersion)
		dockerMIPSBuilderImage    = fmt.Sprintf("%s:%s-mips", dockerBuilderImageName, goVersion)
		dockerS390XBuilderImage   = fmt.Sprintf("%s:%s-s390x", dockerBuilderImageName, goVersion)
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
		case stringInSlice(platform, defaultS390Platforms):
			s390xPlatforms = append(s390xPlatforms, platform)
		case stringInMapKeys(platform, armPlatformsAliases):
			armPlatforms = append(armPlatforms, armPlatformsAliases[platform]...)
		default:
			unknownPlatforms = append(unknownPlatforms, platform)
		}
	}

	if len(unknownPlatforms) > 0 {
		warn(errors.Errorf("unknown/unhandled platforms: %s", unknownPlatforms))
	}

	var pgroups []platformGroup

	if !cgo {
		// In non-CGO, use the base image without any crossbuild toolchain
		var allPlatforms []string
		allPlatforms = append(allPlatforms, mainPlatforms[:]...)
		allPlatforms = append(allPlatforms, armPlatforms[:]...)
		allPlatforms = append(allPlatforms, powerPCPlatforms[:]...)
		allPlatforms = append(allPlatforms, mipsPlatforms[:]...)
		allPlatforms = append(allPlatforms, s390xPlatforms[:]...)

		for _, platform := range allPlatforms {
			name := "base-" + strings.ReplaceAll(platform, "/", "-")
			pgroups = append(pgroups, platformGroup{name, dockerBaseBuilderImage, platform})
		}

		// Pull build image
		err := dockerPull(dockerBaseBuilderImage)
		if err != nil {
			fatal(err)
		}
	} else {
		if len(mainPlatforms) > 0 {
			for _, platform := range mainPlatforms {
				name := "base-" + strings.ReplaceAll(platform, "/", "-")
				pgroups = append(pgroups, platformGroup{name, dockerMainBuilderImage, platform})
			}

			err := dockerPull(dockerMainBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(armPlatforms) > 0 {
			for _, platform := range armPlatforms {
				name := "arm-" + strings.ReplaceAll(platform, "/", "-")
				pgroups = append(pgroups, platformGroup{name, dockerARMBuilderImage, platform})
			}

			err := dockerPull(dockerARMBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(powerPCPlatforms) > 0 {
			for _, platform := range powerPCPlatforms {
				name := "powerpc-" + strings.ReplaceAll(platform, "/", "-")
				pgroups = append(pgroups, platformGroup{name, dockerPowerPCBuilderImage, platform})
			}

			err := dockerPull(dockerPowerPCBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(mipsPlatforms) > 0 {
			for _, platform := range mipsPlatforms {
				name := "mips-" + strings.ReplaceAll(platform, "/", "-")
				pgroups = append(pgroups, platformGroup{name, dockerMIPSBuilderImage, platform})
			}

			err := dockerPull(dockerMIPSBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(s390xPlatforms) > 0 {
			for _, platform := range s390xPlatforms {
				name := "s390x-" + strings.ReplaceAll(platform, "/", "-")
				pgroups = append(pgroups, platformGroup{name, dockerS390XBuilderImage, platform})
			}

			err := dockerPull(dockerS390XBuilderImage)
			if err != nil {
				fatal(err)
			}
		}
	}

	var buildNum int

	// Use CROSSBUILDN for concurrent build number is present
	if len(os.Getenv("CROSSBUILDN")) > 0 {
		buildNum, _ = strconv.Atoi(os.Getenv("CROSSBUILDN"))
	}

	// Use number of CPU - 1 as concurrent build number
	if buildNum == 0 {
		buildNum = int(math.Max(1, float64(runtime.NumCPU())-1))
	}

	sem := make(chan struct{}, buildNum)
	errs := make([]error, 0, len(platforms))

	fmt.Printf("> building %d concurrent crossbuilds\n", buildNum)

	// Launching builds concurrently
	for _, pg := range pgroups {
		sem <- struct{}{}

		go func(pg platformGroup) {
			start := time.Now()
			if err := pg.Build(repoPath); err != nil {
				errs = append(errs, errors.Wrapf(err, "The %s builder docker image exited unexpectedly", pg.Name))
			}
			duration := time.Since(start)
			fmt.Printf("> build %s took %v\n", pg.Name, duration.Round(time.Millisecond))
			<-sem
		}(pg)
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

type platformGroup struct {
	Name        string
	DockerImage string
	Platform    string
}

func dockerPull(image string) error {
	pull := exec.Command("docker", "pull", image)
	err := pull.Run()

	return err
}

func (pg platformGroup) Build(repoPath string) error {
	if len(pg.Platform) == 0 {
		return nil
	}

	fmt.Printf("> running the %s builder docker image\n", pg.Name)

	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "couldn't get current working directory")
	}

	ctrName := "promu-crossbuild-" + pg.Name + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	args := []string{"create", "-t", "--name", ctrName}

	// If we build with a local docker we mount /go/pkg/ to share go mod cache
	if len(os.Getenv("DOCKER_HOST")) == 0 {
		args = append(args, "-v", firstGoPath()+"/pkg/:/go/pkg/")
		args = append(args, "-v", cwd+"/.:/app/")
	}

	args = append(args, pg.DockerImage, "-i", repoPath, "-p", pg.Platform)

	err = sh.RunCommand("docker", args...)
	if err != nil {
		return err
	}

	// Copy source one item at a time to discard the .build dir because docker cp
	// does not honour .dockerignore
	files, err := ioutil.ReadDir("./")
	if err != nil {
		return err
	}

	excludes := []string{
		".build",
	}

	// Only do docker cp if using remote docker
	if len(os.Getenv("DOCKER_HOST")) > 0 {
	FILES:
		for _, file := range files {
			for _, ex := range excludes {
				if file.Name() == ex {
					continue FILES
				}
			}

			var src, dst string

			if !file.IsDir() {
				src = path.Join(cwd, file.Name())
				dst = ctrName + ":" + path.Join("/app", file.Name())
			} else {
				src = path.Join(cwd, file.Name()) + "/"
				dst = ctrName + ":" + path.Join("/app", file.Name()) + "/"
			}

			err = sh.RunCommand("docker", "cp", src, dst)
			if err != nil {
				return err
			}
		}
	}

	// If we build using a remote docker then we cp our local go mod cache
	if len(os.Getenv("DOCKER_HOST")) > 0 {
		err = sh.RunCommand("docker", "cp", firstGoPath()+"/pkg/", ctrName+":/go/pkg/")
		if err != nil {
			return err
		}
	}

	err = sh.RunCommand("docker", "start", "-a", ctrName)
	if err != nil {
		return err
	}

	err = func() error {
		// Avoid doing multiple copy at the same time
		dockerCopyMutex.Lock()
		defer dockerCopyMutex.Unlock()

		return sh.RunCommand("docker", "cp", "-a", ctrName+":/app/.build/.", cwd+"/.build")
	}()

	if err != nil {
		return err
	}

	return sh.RunCommand("docker", "rm", "-f", ctrName)
}

func firstGoPath() string {
	gopath := os.Getenv("GOPATH")

	if strings.Contains(gopath, ":") {
		return gopath[0:strings.Index(gopath, ":")]
	}

	return gopath
}
