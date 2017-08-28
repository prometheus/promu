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
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/prometheus/promu/util/retry"
	"github.com/prometheus/promu/util/sh"
)

var (
	allowedRetries int
)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release [<location>]",
	Short: "Upload all release files to the Github release",
	Long:  `Upload all release files to the Github release`,
	Run: func(cmd *cobra.Command, args []string) {
		runRelease(optArg(args, 0, "."))
	},
}

// init prepares cobra flags
func init() {
	Promu.AddCommand(releaseCmd)

	releaseCmd.Flags().IntVar(&allowedRetries, "retry", 2, "Number of retries to perform when upload fails")
}

func runRelease(location string) {
	if err := filepath.Walk(location, releaseFile); err != nil {
		fatal(errors.Wrap(err, "Failed to upload all files"))
	}
}

func releaseFile(path string, f os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if f.IsDir() {
		return nil
	}

	filename := filepath.Base(path)
	maxAttempts := allowedRetries + 1
	err = retry.Do(func(attempt int) (bool, error) {
		err := uploadFile(filename, path)
		if err != nil {
			time.Sleep(2 * time.Second)
		}
		return attempt < maxAttempts, err
	})
	if err != nil {
		return errors.Wrapf(err, "failed to upload %q after %d attempts", filename, maxAttempts)
	}
	fmt.Println(" > uploaded", filename)

	return nil
}

func uploadFile(filename string, path string) error {
	return sh.RunCommand("github-release", "upload",
		"--user", info.Owner,
		"--repo", info.Name,
		"--tag", fmt.Sprintf("v%s", info.Version),
		"--name", filename,
		"--file", path)
}
