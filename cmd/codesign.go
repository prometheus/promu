// Copyright © 2024 Prometheus Team
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
	"path/filepath"

	"github.com/prometheus/promu/util/sh"
)

var (
	codesigncmd = app.Command("codesign", "Code sign the darwin binary using rcodesign.")
	binaryPath  = codesigncmd.Arg("path", "Absolute path to binary to be signed").Required().String()
)

func runCodeSign(binaryPath string) {
	codeSignGoBinary(binaryPath)
}

func codeSignGoBinary(binaryPath string) {
	var (
		goVersion              = config.Go.Version
		dockerMainBuilderImage = fmt.Sprintf("%s:%s-main", dockerBuilderImageName, goVersion)
		mountPath              = fmt.Sprintf("/%s", filepath.Base(binaryPath))
		mountPathConcat        = fmt.Sprintf("%s:%s", binaryPath, mountPath)
	)
	fmt.Printf("> using rcodesign to sign the binary file at path %s\n", binaryPath)

	// Example:
	// docker run --entrypoint "rcodesign" --rm -v "/path/to/darwin-arm64/node_exporter:/node_exporter"
	// quay.io/prometheus/golang-builder:1.21-main sign /node_exporter
	err := sh.RunCommand("docker", "run", "--entrypoint",
		"rcodesign", "--rm", "-v", mountPathConcat,
		dockerMainBuilderImage, "sign", mountPath)
	if err != nil {
		fmt.Printf("Couldn't sign the binary as intended: %s", err)
	}
}
