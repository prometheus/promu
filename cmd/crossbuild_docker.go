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
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/cache"

	"github.com/prometheus/promu/util/sh"
)

var (
	dockerBuilderImageName = "quay.io/prometheus/golang-builder"
)

var (
	crossbuilddockercmd = app.Command("crossbuild-docker", "Crossbuild a Go project using Golang builder Docker images")

	dockerCopyMutex sync.Mutex
)

func runCrossbuildDocker() {
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
		pgroups = append(pgroups, platformGroup{"base-main", dockerBaseBuilderImage, mainPlatforms})
		pgroups = append(pgroups, platformGroup{"base-arm", dockerBaseBuilderImage, armPlatforms})
		pgroups = append(pgroups, platformGroup{"base-powerpc", dockerBaseBuilderImage, powerPCPlatforms})
		pgroups = append(pgroups, platformGroup{"base-mips", dockerBaseBuilderImage, mipsPlatforms})
		pgroups = append(pgroups, platformGroup{"base-s390x", dockerBaseBuilderImage, s390xPlatforms})

		// Pull build image
		err := dockerPull(dockerBaseBuilderImage)
		if err != nil {
			fatal(err)
		}
	} else {
		if len(mainPlatforms) > 0 {
			pgroups = append(pgroups, platformGroup{"main", dockerMainBuilderImage, mainPlatforms})

			err := dockerPull(dockerMainBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(armPlatforms) > 0 {
			pgroups = append(pgroups, platformGroup{"arm", dockerARMBuilderImage, armPlatforms})

			err := dockerPull(dockerARMBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(powerPCPlatforms) > 0 {
			pgroups = append(pgroups, platformGroup{"powerpc", dockerPowerPCBuilderImage, powerPCPlatforms})

			err := dockerPull(dockerPowerPCBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(mipsPlatforms) > 0 {
			pgroups = append(pgroups, platformGroup{"mips", dockerMIPSBuilderImage, mipsPlatforms})

			err := dockerPull(dockerMIPSBuilderImage)
			if err != nil {
				fatal(err)
			}
		}

		if len(s390xPlatforms) > 0 {
			pgroups = append(pgroups, platformGroup{"s390x", dockerS390XBuilderImage, s390xPlatforms})

			err := dockerPull(dockerS390XBuilderImage)
			if err != nil {
				fatal(err)
			}
		}
	}

	sem := make(chan struct{}, *crossbuildJobs)
	errs := make([]error, 0, len(platforms))

	fmt.Printf("~ building up to %d concurrent crossbuilds\n", *crossbuildJobs)
	fmt.Printf("~ building up to %d concurrent binaries\n", *binaryJobs)

	// Launching builds concurrently
	for _, pg := range pgroups {
		sem <- struct{}{}

		go func(pg platformGroup) {
			fmt.Printf("< building %s\n", pg.Name)

			start := time.Now()
			if err := pg.Build(repoPath); err != nil {
				errs = append(errs, errors.Wrapf(err, "The %s builder docker image exited unexpectedly", pg.Name))
			}
			duration := time.Since(start)

			fmt.Printf("> %s took (build in %v)\n", pg.Name, duration.Round(time.Millisecond))
			<-sem
		}(pg)
	}

	// Wait for builds to finish
	for {
		if len(sem) != 0 {
			time.Sleep(100 * time.Millisecond)
		} else {
			break
		}
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
	Platforms   []string
}

func dockerPull(image string) error {
	pull := exec.Command("docker", "pull", image)
	err := pull.Run()

	return err
}

const localGoCacheDir = ".cache/go-build"
const containerGoCacheDir = "/go/.cache/go-build"

func (pg platformGroup) Build(repoPath string) error {
	if len(pg.Platforms) == 0 {
		return nil
	}

	fmt.Printf(" > running the %s builder docker image\n", pg.Name)

	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "couldn't get current working directory")
	}

	ctrName := "promu-crossbuild-" + pg.Name + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	args := []string{"create", "-t", "--name", ctrName}

	// If we build with a local docker we mount /go/pkg/ to share go mod cache
	if len(os.Getenv("DOCKER_HOST")) == 0 {
		_, err := os.Stat(localGoCacheDir)
		if err != nil {
			os.MkdirAll(localGoCacheDir, 0755)
		}

		args = append(args, "-v", firstGoPath()+"/pkg/:/go/pkg/")
		args = append(args, "-v", cwd+"/.:/app/")

		args = append(args, "-v", cache.DefaultDir()+"/:"+containerGoCacheDir+"/")
		args = append(args, "--env", "GOCACHE="+containerGoCacheDir)
	}

	args = append(args, pg.DockerImage, "-i", repoPath, "-p", strings.Join(pg.Platforms, " "))

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
		".cache",
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

	// If we build using a remote docker then we cp the result of the build
	if len(os.Getenv("DOCKER_HOST")) > 0 {
		err = func() error {
			// Avoid doing multiple copy at the same time
			dockerCopyMutex.Lock()
			defer dockerCopyMutex.Unlock()

			return sh.RunCommand("docker", "cp", "-a", ctrName+":/app/.build/.", cwd+"/.build")
		}()

		if err != nil {
			return err
		}
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
