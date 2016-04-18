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
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Print info about current project and exit",
	Long:  `Print info about current project and exit`,
	Run: func(cmd *cobra.Command, args []string) {
		runInfo()
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(infoCmd)
}

type ProjectInfo struct {
	Branch   string
	Name     string
	Owner    string
	Repo     string
	Revision string
	Version  string
}

func NewProjectInfo() ProjectInfo {
	repo := repoLocation()
	version, err := findVersion()
	if err != nil {
		fatalMsg("Unable to find project's version", err)
	}
	return ProjectInfo{
		Branch:   shellOutput("git rev-parse --abbrev-ref HEAD"),
		Name:     filepath.Base(repo),
		Owner:    filepath.Base(filepath.Dir(repo)),
		Repo:     repo,
		Revision: shellOutput("git rev-parse --short HEAD"),
		Version:  version,
	}
}

func runInfo() {
	fmt.Println("Name:", info.Name)
	fmt.Println("Version:", info.Version)
	fmt.Println("Owner:", info.Owner)
	fmt.Println("Repo:", info.Repo)
	fmt.Println("Branch:", info.Branch)
	fmt.Println("Revision:", info.Revision)
}

func repoLocation() string {
	repo := shellOutput("git config --get remote.origin.url")
	repo = strings.TrimPrefix(repo, "http://")
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimPrefix(repo, "git@")
	repo = strings.TrimSuffix(repo, ".git")
	return strings.Replace(repo, ":", "/", -1)
}

func findVersion() (string, error) {
	var files = []string{"VERSION", "version/VERSION"}
	for _, file := range files {
		if fileExists(file) {
			return readFile(file), nil
		}
	}
	return "", errors.New("missing `VERSION` or `version/VERSION` file")
}
