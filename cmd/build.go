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
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/promu/util/sh"
)

var (
	buildcmd = app.Command("build", "Build a Go project").
			Action(func(c *kingpin.ParseContext) error {
			return hasRequiredConfigurations("repository.path")
		})
	buildCgoFlagSet bool
	buildCgoFlag    = buildcmd.Flag("cgo", "Enable CGO").
			PreAction(func(c *kingpin.ParseContext) error {
			buildCgoFlagSet = true
			return nil
		}).Bool()
	prefixFlagSet bool
	prefixFlag    = buildcmd.Flag("prefix", "Specific dir to store binaries (default is .)").
			PreAction(func(c *kingpin.ParseContext) error {
			prefixFlagSet = true
			return nil
		}).String()
	binaries = buildcmd.Arg("binary-names", "List of binaries to build").Default("all").Strings()
)

// Binary represents a built binary.
type Binary struct {
	Name string
	Path string
}

// Check if binary names passed to build command are in the config.
// Returns an array of Binary to build, or error.
func validateBinaryNames(binaryNames []string, cfgBinaries []Binary) ([]Binary, error) {
	var binaries []Binary

OUTER:
	for _, binaryName := range binaryNames {
		for _, binary := range cfgBinaries {
			if binaryName == binary.Name {
				binaries = append(binaries, binary)
				continue OUTER
			}
		}
		return nil, fmt.Errorf("binary %s not found in config", binaryName)
	}
	return binaries, nil
}

func buildBinary(ext string, prefix string, ldflags string, binary Binary) {
	binaryName := fmt.Sprintf("%s%s", binary.Name, ext)
	fmt.Printf(" >   %s\n", binaryName)

	repoPath := viper.GetString("repository.path")
	flags := viper.GetString("build.flags")

	params := []string{"build",
		"-o", path.Join(prefix, binaryName),
		"-ldflags", ldflags,
	}

	params = append(params, sh.SplitParameters(flags)...)
	params = append(params, path.Join(repoPath, binary.Path))
	if err := sh.RunCommand("go", params...); err != nil {
		fatal(errors.Wrap(err, "command failed: "+strings.Join(params, " ")))
	}
}

func buildAll(ext string, prefix string, ldflags string, binaries []Binary) {
	for _, binary := range binaries {
		buildBinary(ext, prefix, ldflags, binary)
	}
}

func bindViperBuildFlags() {
	if prefixFlagSet {
		viperPrefixFlag := ViperFlagValue{"prefix", *prefixFlag, "string", true}
		viper.BindFlagValue("build.prefix", viperPrefixFlag)
	}
	if buildCgoFlagSet {
		viperCgoFlag := ViperFlagValue{"cgo", strconv.FormatBool(*buildCgoFlag), "bool", true}
		viper.BindFlagValue("go.cgo", viperCgoFlag)
	}
}

func runBuild(binariesString string) {
	bindViperBuildFlags()
	var (
		cgo    = viper.GetBool("go.cgo")
		prefix = viper.GetString("build.prefix")

		ext      string
		binaries []Binary
		ldflags  string
	)

	if goos == "windows" {
		ext = ".exe"
	}

	ldflags = getLdflags(info)

	os.Setenv("CGO_ENABLED", "0")
	if cgo {
		os.Setenv("CGO_ENABLED", "1")
	}
	defer os.Unsetenv("CGO_ENABLED")

	if err := viper.UnmarshalKey("build.binaries", &binaries); err != nil {
		fatal(errors.Wrap(err, "Failed to Unmashal binaries"))
	}

	if binariesString == "all" {
		buildAll(ext, prefix, ldflags, binaries)
		return
	}

	binariesArray := strings.Split(binariesString, ",")
	binariesToBuild, err := validateBinaryNames(binariesArray, binaries)
	if err != nil {
		fatal(errors.Wrap(err, "validation of given binary names for build command failed"))
	}

	for _, binary := range binariesToBuild {
		buildBinary(ext, prefix, ldflags, binary)
	}
}

func getLdflags(info ProjectInfo) string {
	var ldflags []string

	if viper.IsSet("build.ldflags") {
		var (
			tmplOutput = new(bytes.Buffer)
			fnMap      = template.FuncMap{
				"date":     time.Now().UTC().Format,
				"host":     os.Hostname,
				"repoPath": RepoPathFunc,
				"user":     UserFunc,
			}
			ldflagsTmpl = viper.GetString("build.ldflags")
		)

		tmpl, err := template.New("ldflags").Funcs(fnMap).Parse(ldflagsTmpl)
		if err != nil {
			fatal(errors.Wrap(err, "Failed to parse ldflags text/template"))
		}

		if err := tmpl.Execute(tmplOutput, info); err != nil {
			fatal(errors.Wrap(err, "Failed to execute ldflags text/template"))
		}

		ldflags = append(ldflags, strings.Split(tmplOutput.String(), "\n")...)
	} else {
		ldflags = append(ldflags, fmt.Sprintf("-X main.Version=%s", info.Version))
	}

	staticBinary := viper.GetBool("build.static")
	if staticBinary && goos != "darwin" && goos != "solaris" && !stringInSlice(`-extldflags '-static'`, ldflags) {
		ldflags = append(ldflags, `-extldflags '-static'`)
	}

	return strings.Join(ldflags[:], " ")
}

// UserFunc returns the current username.
func UserFunc() (interface{}, error) {
	// os/user.Current() doesn't always work without CGO
	return shellOutput("whoami"), nil
}

// RepoPathFunc returns the repository path.
func RepoPathFunc() interface{} {
	return viper.GetString("repository.path")
}
