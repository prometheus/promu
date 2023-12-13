// Copyright © 2019 Prometheus Team
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

package repository

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
)

// Info represents current project useful information.
type Info struct {
	Branch   string
	Name     string
	Owner    string
	Repo     string
	Revision string
	Version  string
}

// shellOutput executes a shell command and returns the trimmed output.
func shellOutput(cmd string, arg ...string) string {
	out, _ := shellOutputWithError(cmd, arg...)
	return out
}

// shellOutputWithError executes a shell command and returns the trimmed output and error.
func shellOutputWithError(cmd string, arg ...string) (string, error) {
	out, err := exec.Command(cmd, arg...).Output()
	return strings.Trim(string(out), " \n\r"), err
}

// NewInfo returns a new Info.
func NewInfo(warnf func(error)) (Info, error) {
	if warnf == nil {
		warnf = func(error) {}
	}

	var (
		info Info
		err  error
	)

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		// Not a git repository.
		repo, err := os.Getwd()
		if err != nil {
			return info, fmt.Errorf("couldn't get current working directory: %w", err)
		}
		repo = strings.TrimPrefix(repo, os.Getenv("GOPATH"))
		repo = strings.TrimPrefix(repo, "/src/")

		user, err := user.Current()
		if err != nil {
			return info, fmt.Errorf("couldn't get current user: %w", err)
		}

		info = Info{
			Branch:   "non-git",
			Name:     filepath.Base(repo),
			Owner:    user.Username,
			Repo:     repo,
			Revision: "non-git",
		}
	} else {
		branch, err := shellOutputWithError("git", "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return info, fmt.Errorf("unable to get the current branch: %w", err)
		}

		remote, err := shellOutputWithError("git", "config", "--get", fmt.Sprintf("branch.%s.remote", branch))
		if err != nil {
			// default to origin.
			remote = "origin"
		}

		repoURL, err := shellOutputWithError("git", "config", "--get", fmt.Sprintf("remote.%s.url", remote))
		if err != nil {
			warnf(fmt.Errorf("unable to get repository location for remote %q: %w", remote, err))
		}
		repo, err := repoLocation(repoURL)
		if err != nil {
			return info, fmt.Errorf("couldn't parse repository location: %q: %w", repoURL, err)
		}
		info = Info{
			Branch:   branch,
			Name:     filepath.Base(repo),
			Owner:    filepath.Base(filepath.Dir(repo)),
			Repo:     repo,
			Revision: shellOutput("git", "rev-parse", "HEAD"),
		}
	}

	info.Version, err = findVersion()
	if err != nil {
		warnf(fmt.Errorf("unable to find project's version: %w", err))
	}
	return info, nil
}

// Convert SCP-like URL to SSH URL(e.g. [user@]host.xz:path/to/repo.git/)
// ref. http://git-scm.com/docs/git-fetch#_git_urls
// (golang hasn't supported Perl-like negative look-behind match)
var (
	hasSchemePattern  = regexp.MustCompile("^[^:]+://")
	scpLikeURLPattern = regexp.MustCompile("^([^@]+@)?([^:]+):/?(.+)$")
)

func repoLocation(repo string) (string, error) {
	if !hasSchemePattern.MatchString(repo) && scpLikeURLPattern.MatchString(repo) {
		matched := scpLikeURLPattern.FindStringSubmatch(repo)
		user := matched[1]
		host := matched[2]
		path := matched[3]
		repo = fmt.Sprintf("ssh://%s%s/%s", user, host, path)
	}

	u, err := url.Parse(repo)
	if err != nil {
		return "", err
	}

	repo = fmt.Sprintf("%s%s", strings.Split(u.Host, ":")[0], u.Path)
	repo = strings.TrimSuffix(repo, ".git")
	return repo, nil
}

func findVersion() (string, error) {
	for _, file := range []string{"VERSION", "version/VERSION"} {
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		return strings.Trim(string(b), "\n\r "), nil
	}

	return strings.TrimPrefix(shellOutput("git", "describe", "--tags", "--always", "--dirty"), "v"), nil
}

// ToSemver returns a *semver.Version from Info.
func (i Info) ToSemver() (*semver.Version, error) {
	if strings.HasPrefix(i.Version, "v") {
		return nil, fmt.Errorf("version %q shouldn't start with 'v'", i.Version)
	}
	semVer, err := semver.NewVersion(i.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid semver version: %w", err)
	}
	return semVer, nil
}
