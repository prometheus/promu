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

package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v25/github"
	"github.com/pkg/errors"

	"github.com/prometheus/promu/pkg/changelog"
	githubUtil "github.com/prometheus/promu/pkg/github"
)

var (
	bumpcmd = app.Command("bump", "Update CHANGELOG.md and VERSION files to the next version")

	bumpLevel      = bumpcmd.Flag("level", "Level of version to increment (should be one of major, minor, patch, pre)").Default("minor").Enum("major", "minor", "patch", "pre")
	bumpPreRelease = bumpcmd.Flag("pre-release", "Pre-release identifier").Default("rc.0").String()
	bumpBaseBranch = bumpcmd.Flag("base-branch", "Base branch").Default("master").String()
	bumpDryRun     = bumpcmd.Flag("dry-run", "Do not modify the files").Bool()
)

type pullRequest struct {
	Number      int
	Title       string
	Contributor string
	Kinds       changelog.Kinds
}

var (
	labelPrefix = "changelog/"
	skipLabel   = labelPrefix + "skip"
)

type changelogData struct {
	Version      string
	Date         string
	PullRequests []pullRequest
	Skipped      []pullRequest
	Contributors []string
}

const changelogTmpl = `## {{ .Version }} / {{ .Date }}
{{ range .PullRequests }}
* [{{ .Kinds.String }}] {{ makeSentence .Title }} #{{ .Number }}
{{- end }}
<!-- Unclassified pull requests:{{ range .Skipped }}
* [{{ .Kinds.String }}] {{ makeSentence .Title }} #{{ .Number }}
{{- end }} -->

Contributors:
{{ range .Contributors }}
* @{{ . }}
{{- end }}

`

func writeChangelog(w io.Writer, version string, prs []pullRequest) error {
	var (
		visible, hidden  []pullRequest
		uniqContributors = map[string]struct{}{"": struct{}{}}
		contributors     []string
	)
	for _, pr := range prs {
		if len(pr.Kinds) > 0 {
			visible = append(visible, pr)
		} else {
			hidden = append(hidden, pr)
		}

		if _, ok := uniqContributors[pr.Contributor]; ok {
			continue
		}
		uniqContributors[pr.Contributor] = struct{}{}
		contributors = append(contributors, pr.Contributor)
	}
	sort.SliceStable(visible, func(i int, j int) bool { return visible[i].Kinds.Before(visible[j].Kinds) })
	sort.SliceStable(hidden, func(i int, j int) bool { return hidden[i].Kinds.Before(hidden[j].Kinds) })
	sort.Strings(contributors)

	tmpl, err := template.New("changelog").Funcs(
		template.FuncMap{
			"makeSentence": func(s string) string {
				s = strings.TrimRight(s, ".")
				return s + "."
			},
		}).Parse(changelogTmpl)
	if err != nil {
		return errors.Wrap(err, "invalid template")
	}

	return tmpl.Execute(w, &changelogData{
		Version:      version,
		Date:         time.Now().Format("2006-01-02"),
		PullRequests: visible,
		Skipped:      hidden,
		Contributors: contributors,
	})
}

func runBumpVersion(changelogPath, versionPath string, bumpLevel string, preRelease string, baseBranch string, dryRun bool) error {
	current, err := projInfo.ToSemver()
	if err != nil {
		return err
	}

	next := *current
	switch bumpLevel {
	case "major":
		next = current.IncMajor()
	case "minor":
		next = current.IncMinor()
	case "patch":
		next = current.IncPatch()
	}
	next, err = next.SetPrerelease(preRelease)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if *timeout != time.Duration(0) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}
	client, err := githubUtil.NewClient(ctx)
	if err != nil {
		info(fmt.Sprintf("failed to create authenticated GitHub client: %v", err))
		info("fallback to GitHub client without unauthentication")
		client = github.NewClient(nil)
	}

	lastTag := "v" + current.String()
	commit, _, err := client.Repositories.GetCommit(ctx, projInfo.Owner, projInfo.Name, lastTag)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Fail to get the GitHub commit for %s", lastTag))
	}
	lastTagTime := commit.GetCommit().GetCommitter().GetDate()
	lastCommitSHA := commit.GetSHA()

	// Gather all pull requests merged since the last tag.
	var ghPrs []*github.PullRequest
	err = githubUtil.ReadAll(
		func(opts *github.ListOptions) (*github.Response, error) {
			prs, resp, err := client.PullRequests.List(ctx, projInfo.Owner, projInfo.Name, &github.PullRequestListOptions{
				State:       "closed",
				Sort:        "updated",
				Direction:   "desc",
				ListOptions: *opts,
			})
			if err != nil {
				return nil, errors.Wrap(err, "Fail to list GitHub pull requests")
			}
			for _, pr := range prs {
				if pr.GetUpdatedAt().Before(lastTagTime) {
					// We've reached pull requests that haven't changed since
					// the reference tag so we can stop now.
					return nil, nil
				}
				if pr.GetBase().GetRef() != baseBranch {
					continue
				}
				if pr.GetMergedAt().IsZero() || pr.GetMergedAt().Before(lastTagTime) {
					continue
				}
				if pr.GetMergeCommitSHA() == lastCommitSHA {
					continue
				}
				ghPrs = append(ghPrs, pr)
			}
			return resp, nil
		},
	)
	if err != nil {
		return err
	}

	// Extract information from pull requests.
	var prs []pullRequest
	for _, pr := range ghPrs {
		var (
			kinds       changelog.Kinds
			contributor string
		)
		for _, lbl := range pr.Labels {
			s := strings.TrimPrefix(strings.ToLower(lbl.GetName()), "kind/")
			if k, err := changelog.FromString(s); err == nil {
				kinds = append(kinds, k)
			}
		}
		if pr.GetUser() != nil {
			contributor = pr.GetUser().GetLogin()
		}
		prs = append(prs, pullRequest{
			Kinds:       kinds,
			Number:      pr.GetNumber(),
			Title:       pr.GetTitle(),
			Contributor: contributor,
		})
	}

	if dryRun {
		return writeChangelog(os.Stdout, next.String(), prs)
	}

	// Update the changelog file.
	original, err := ioutil.ReadFile(changelogPath)
	if err != nil {
		return err
	}
	f, err := os.Create(changelogPath)
	if err != nil {
		return err
	}
	defer f.Close()
	err = writeChangelog(f, next.String(), prs)
	if err != nil {
		return err
	}
	_, err = f.Write(original)
	if err != nil {
		return err
	}

	// Update the version file (if present).
	if _, err = os.Stat(versionPath); err == nil {
		f, err := os.Create(versionPath)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.WriteString(next.String())
		if err != nil {
			return err
		}
	}

	return nil
}
