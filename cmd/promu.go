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
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	buildContext = build.Default
	goos         = buildContext.GOOS
	goarch       = buildContext.GOARCH

	cfgFile  string
	info     ProjectInfo
	useViper bool
	verbose  bool
)

// Promu represents the base command when called without any subcommands
var Promu = &cobra.Command{
	Use:           "promu",
	Short:         "promu is the utility tool for Prometheus projects",
	Long:          `promu is the utility tool for Prometheus projects`,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		info, err = NewProjectInfo()
		if err != nil {
			return err
		}
		return initConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := Promu.Execute(); err != nil {
		printErr(err)
		os.Exit(2)
	}
}

// init prepares cobra flags
func init() {
	Promu.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is ./.promu.yml)")
	Promu.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	Promu.PersistentFlags().BoolVar(&useViper, "viper", true, "Use Viper for configuration")

	viper.BindPFlag("useViper", Promu.PersistentFlags().Lookup("viper"))
	viper.BindPFlag("verbose", Promu.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	if useViper != true {
		return nil
	}
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".promu") // name of config file (without extension)
	viper.AddConfigPath(".")      // look for config in the working directory
	viper.SetEnvPrefix("promu")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "Error in config file")
	}
	if verbose {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	setDefaultConfigValues()
	return nil
}

func setDefaultConfigValues() {
	if !viper.IsSet("build.binaries") {
		binaries := []map[string]string{{"name": info.Name, "path": "."}}
		viper.Set("build.binaries", binaries)
	}
	if !viper.IsSet("build.prefix") {
		viper.Set("build.prefix", ".")
	}
	if !viper.IsSet("crossbuild.platforms") {
		platforms := defaultMainPlatforms
		platforms = append(platforms, defaultARMPlatforms...)
		platforms = append(platforms, defaultPowerPCPlatforms...)
		platforms = append(platforms, defaultMIPSPlatforms...)
		viper.Set("crossbuild.platforms", platforms)
	}
	if !viper.IsSet("tarball.prefix") {
		viper.Set("tarball.prefix", ".")
	}
	if !viper.IsSet("go.version") {
		viper.Set("go.version", "1.8.3")
	}
	if !viper.IsSet("go.cgo") {
		viper.Set("go.cgo", false)
	}
	if !viper.IsSet("repository.path") {
		viper.Set("repository.path", info.Repo)
	}
}

// warn prints a non-fatal error
func warn(err error) {
	if verbose {
		fmt.Fprintf(os.Stderr, `/!\ %+v\n`, err)
	} else {
		fmt.Fprintln(os.Stderr, `/!\`, err)
	}
}

// printErr prints a error
func printErr(err error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "!! %+v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "!!", err)
	}
}

// fatal prints a error and exit
func fatal(err error) {
	printErr(err)
	os.Exit(1)
}

// shellOutput executes a shell command and return the trimmed output
func shellOutput(cmd string) string {
	args := strings.Fields(cmd)
	out, _ := exec.Command(args[0], args[1:]...).Output()
	return strings.Trim(string(out), " \n\r")
}

// fileExists checks if a file exists
func fileExists(path ...string) bool {
	finfo, err := os.Stat(filepath.Join(path...))
	if err == nil && !finfo.IsDir() {
		return true
	}
	if os.IsNotExist(err) || finfo.IsDir() {
		return false
	}
	if err != nil {
		fatal(err)
	}
	return true
}

// readFile reads a file and return the trimmed output
func readFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.Trim(string(data), "\n\r ")
}

func optArg(args []string, i int, def string) string {
	if i+1 > len(args) {
		return def
	}
	return args[i]
}

func envOr(name, def string) string {
	s := os.Getenv(name)
	if s == "" {
		return def
	}
	return s
}

func stringInSlice(needle string, haystack []string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

func stringInMapKeys(needle string, haystack map[string][]string) bool {
	_, ok := haystack[needle]
	return ok
}

func hasRequiredConfigurations(configVars ...string) error {
	for _, configVar := range configVars {
		if !viper.IsSet(configVar) {
			return errors.Errorf("missing required '%s' configuration", configVar)
		}
	}
	return nil
}
