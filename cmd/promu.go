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
	"github.com/spf13/viper"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// ViperFlagValue defines a flag wrapper for Viper integration
type ViperFlagValue struct {
	name        string
	valueString string
	typeString  string
	changed     bool
}

// Name of the flag
func (f ViperFlagValue) Name() string { return f.name }

// ValueString the value of the flag in string form
func (f ViperFlagValue) ValueString() string { return f.valueString }

// ValueType the type (int, bool, etc) of the flag in string form
func (f ViperFlagValue) ValueType() string { return f.typeString }

// HasChanged whether the flag was set on the CLI
func (f ViperFlagValue) HasChanged() bool { return f.changed }

var (
	buildContext = build.Default
	goos         = buildContext.GOOS
	goarch       = buildContext.GOARCH

	cfgFile  string
	info     ProjectInfo
	useViper bool
	verbose  bool

	// app represents the base command
	app = kingpin.New("promu", "promu is the utility tool for Prometheus projects").
		Action(func(ctx *kingpin.ParseContext) error {
			var err error
			info, err = NewProjectInfo()
			if err != nil {
				return err
			}
			return initConfig()
		})
)

// init prepares flags
func init() {
	app.HelpFlag.Short('h')

	// verbose, config, and viper flags cannot be initialized in var section because
	// the app PreAction is dependent on them, and it would create an init loop
	app.Flag("verbose", "Verbose output").Short('v').
		PreAction(func(c *kingpin.ParseContext) error {
			viper.BindFlagValue("verbose", ViperFlagValue{"verbose", "true", "bool", true})
			return nil
		}).BoolVar(&verbose)
	app.Flag("config", "Path to config file").StringVar(&cfgFile)
	app.Flag("viper", "Use Viper for configuration (default true)").Default("true").BoolVar(&useViper)
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case buildcmd.FullCommand():
		runBuild(optArg(*binaries, 0, "all"))
	case checkLicensescmd.FullCommand():
		runCheckLicenses(optArg(*checkLicLocation, 0, "."), *headerLength, *sourceExtensions)
	case checksumcmd.FullCommand():
		runChecksum(optArg(*checksumLocation, 0, "."))
	case crossbuildcmd.FullCommand():
		runCrossbuild()
	case infocmd.FullCommand():
		runInfo()
	case releasecmd.FullCommand():
		runRelease(optArg(*releaseLocation, 0, "."))
	case tarballcmd.FullCommand():
		runTarball(optArg(*tarBinariesLocation, 0, "."))
	case versioncmd.FullCommand():
		runVersion()
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	if !useViper {
		return nil
	}
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".promu") // name of config file (without extension)
	viper.AddConfigPath(".")      // look for config in the working directory
	viper.SetEnvPrefix("promu")
	viper.AutomaticEnv()                                   // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // replacing dots in key names with '_'

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
	if !viper.IsSet("build.static") {
		viper.Set("build.static", true)
	}
	if !viper.IsSet("crossbuild.platforms") {
		platforms := defaultMainPlatforms
		platforms = append(platforms, defaultARMPlatforms...)
		platforms = append(platforms, defaultPowerPCPlatforms...)
		platforms = append(platforms, defaultMIPSPlatforms...)
		platforms = append(platforms, defaultS390Platforms...)
		viper.Set("crossbuild.platforms", platforms)
	}
	if !viper.IsSet("tarball.prefix") {
		viper.Set("tarball.prefix", ".")
	}
	if !viper.IsSet("go.version") {
		viper.Set("go.version", "1.11")
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
