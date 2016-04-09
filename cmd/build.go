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
	"os/user"
	"text/template"
	"time"

	shell "github.com/progrium/go-shell"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a Go project",
	Long:  `Build a Go project`,
	Run: func(cmd *cobra.Command, args []string) {
		runBuild()
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet("build.prefix") {
			viper.Set("build.prefix", ".")
		}
		if !viper.IsSet("build.binaries") {
			binaries := []map[string]string{{"name": info.Name, "path": "."}}
			viper.Set("build.binaries", binaries)
		}
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(buildCmd)

	buildCmd.Flags().String("prefix", "", "Specific dir to store binaries (default is .)")

	viper.BindPFlag("build.prefix", buildCmd.Flags().Lookup("prefix"))
}

type Binary struct {
	Name string
	Path string
}

func runBuild() {
	defer shell.ErrExit()
	shell.Tee = os.Stdout

	if viper.GetBool("verbose") {
		shell.Trace = true
	}

	var (
		prefix   = viper.GetString("build.prefix")
		repoPath = viper.GetString("repository.path")
		flags    = viper.GetString("build.flags")

		ext      string
		binaries []Binary
		ldflags  string
	)

	if goos == "windows" {
		ext = ".exe"
	}

	ldflags = getLdflags(info)
	ldflag := fmt.Sprintf("-ldflags \"%s\"", ldflags)

	os.Setenv("GO15VENDOREXPERIMENT", "1")

	err := viper.UnmarshalKey("build.binaries", &binaries)
	fatalMsg(err, "Failed to Unmashal binaries")

	for _, binary := range binaries {
		binaryName := fmt.Sprintf("%s%s", binary.Name, ext)
		fmt.Printf(" >   %s\n", binaryName)
		sh("go build", flags, ldflag, "-o", shell.Path(prefix, binaryName), shell.Path(repoPath, binary.Path))
	}
}

func getLdflags(info ProjectInfo) string {
	if viper.IsSet("build.ldflags") {
		var (
			tmplOutput = new(bytes.Buffer)
			fnMap      = template.FuncMap{
				"date":     time.Now().UTC().Format,
				"host":     os.Hostname,
				"repoPath": RepoPathFunc,
				"user":     UserFunc,
			}
			ldflags = viper.GetString("build.ldflags")
		)

		tmpl, err := template.New("ldflags").Funcs(fnMap).Parse(ldflags)
		fatalMsg(err, "Failed to parse ldflags text/template")

		err = tmpl.Execute(tmplOutput, info)
		fatalMsg(err, "Failed to execute ldflags text/template")

		if goos != "darwin" {
			tmplOutput.WriteString("-extldflags \"-static\"")
		}

		return tmplOutput.String()
	}

	return fmt.Sprintf("-X main.Version=%s", info.Version)
}

func UserFunc() (interface{}, error) {
	user, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("Failed to get current user : %s", err)
	}
	return user.Username, nil
}

func RepoPathFunc() interface{} {
	return viper.GetString("repository.path")
}
