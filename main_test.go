// Copyright © 2018 Prometheus Team
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

package main

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const (
	examplesDir    = "doc/examples"
	testOutputDir  = "testoutput"
	promuCmdBinary = "promu"
)

var (
	goos   = build.Default.GOOS
	goarch = build.Default.GOARCH

	promuBinaryRelPath      = path.Join(testOutputDir, promuCmdBinary)
	promuBinaryAbsPath, _   = filepath.Abs(promuBinaryRelPath)
	promuExamplesBasic      = path.Join(examplesDir, "basic")
	promuExamplesCrossbuild = path.Join(examplesDir, "crossbuild")
	promuExamplesTarball    = path.Join(examplesDir, "tarball")
)

func TestMain(m *testing.M) {
	setup()
	result := m.Run()
	os.Exit(result)
}

// setup any prerequisites for the tests
func setup() {
	err := os.Mkdir(examplesDir, os.ModePerm)
	if !os.IsExist(err) && err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", promuBinaryAbsPath)
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err, string(output))
	}
}

func errcheck(t *testing.T, err error, output string) {
	if err != nil {
		log.Print(output)
		t.Error(err)
	}
}

func assertTrue(t *testing.T, cond bool) {
	if !cond {
		t.Error("condition isn't true")
	}
}

func assertFileExists(t *testing.T, filepath string) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		log.Print("file does not exist: ", filepath)
		t.Error(err)
	}
}

func createSymlink(t *testing.T, target, newlink string) {
	if _, err := os.Stat(newlink); os.IsNotExist(err) {
		err = os.Symlink(target, newlink)
		errcheck(t, err, "Unable to create symlink "+newlink)
	}
}

func dockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

func TestPromuInfo(t *testing.T) {
	cmd := exec.Command(promuBinaryAbsPath, "info")
	output, err := cmd.CombinedOutput()
	errcheck(t, err, string(output))
	if !strings.HasPrefix(string(output), "Name: promu") {
		t.Error("incorrect output for 'info' command: ", string(output))
	}
}

func TestPromuBuild_Basic(t *testing.T) {
	outputDir := path.Join(testOutputDir, "basic")
	err := os.MkdirAll(outputDir, os.ModePerm)
	errcheck(t, err, "Unable to create output dir")

	createSymlink(t, path.Join("..", "..", promuExamplesBasic, ".promu.yml"),
		path.Join(outputDir, ".promu.yml"))

	cmd := exec.Command(promuBinaryAbsPath, "build")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	errcheck(t, err, string(output))

	assertFileExists(t, path.Join(outputDir, "basic-example"))
}

func TestPromuBuild_AltConfig(t *testing.T) {
	outputDir := path.Join(testOutputDir, "altconf")
	promuConfig := path.Join(promuExamplesBasic, "alt-promu.yml")
	cmd := exec.Command(promuBinaryAbsPath, "build", "--config", promuConfig, "--prefix", outputDir)
	output, err := cmd.CombinedOutput()
	errcheck(t, err, string(output))

	assertFileExists(t, path.Join(outputDir, "alt-basic-example"))
}

func TestPromuBuild_ExtLDFlags(t *testing.T) {
	outputDir := path.Join(testOutputDir, "extldflags")
	promuConfig := path.Join(promuExamplesBasic, "extldflags.yml")
	cmd := exec.Command(promuBinaryAbsPath, "build", "-v", "--config", promuConfig, "--prefix", outputDir)
	output, err := cmd.CombinedOutput()
	assertTrue(t, strings.Contains(string(output), "-extldflags '-ltesting -ltesting01 -static'"))
	errcheck(t, err, string(output))
	assertFileExists(t, path.Join(outputDir, "extldflags"))
}

func TestTarball(t *testing.T) {
	outputDir := path.Join(testOutputDir, "tarball")
	err := os.MkdirAll(outputDir, os.ModePerm)
	errcheck(t, err, "Unable to create output dir")

	createSymlink(t, path.Join("..", "..", promuExamplesBasic, ".promu.yml"),
		path.Join(outputDir, ".promu.yml"))
	createSymlink(t, path.Join("..", "..", promuExamplesTarball, "README.md"),
		path.Join(outputDir, "README.md"))
	createSymlink(t, path.Join("..", "..", promuExamplesTarball, "VERSION"),
		path.Join(outputDir, "VERSION"))

	cmd := exec.Command(promuBinaryAbsPath, "build")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	errcheck(t, err, string(output))

	cmd = exec.Command(promuBinaryAbsPath, "tarball")
	cmd.Dir = outputDir
	output, err = cmd.CombinedOutput()
	errcheck(t, err, string(output))

	tarfileName := fmt.Sprintf("promu-0.1.%s-%s.tar.gz", goos, goarch)
	assertFileExists(t, path.Join(outputDir, tarfileName))
}

func TestPromuCrossbuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping crossbuild test in short mode.")
	}
	if !dockerAvailable() {
		t.Error("unable to connect to docker daemon.")
		return
	}

	cmd := exec.Command(promuBinaryAbsPath, "crossbuild")
	cmd.Dir = promuExamplesCrossbuild
	output, err := cmd.CombinedOutput()
	errcheck(t, err, string(output))
	assertFileExists(t, path.Join(promuExamplesCrossbuild, ".build", "linux-386", "crossbuild-example"))
	assertFileExists(t, path.Join(promuExamplesCrossbuild, ".build", "linux-amd64", "crossbuild-example"))

	cmd = exec.Command(promuBinaryAbsPath, "crossbuild", "tarballs")
	cmd.Dir = promuExamplesCrossbuild
	defer os.RemoveAll(path.Join(promuExamplesCrossbuild, ".tarballs"))
	output, err = cmd.CombinedOutput()
	errcheck(t, err, string(output))
	assertFileExists(t, path.Join(promuExamplesCrossbuild, ".tarballs", "promu-0.1.linux-386.tar.gz"))
	assertFileExists(t, path.Join(promuExamplesCrossbuild, ".tarballs", "promu-0.1.linux-amd64.tar.gz"))
}
