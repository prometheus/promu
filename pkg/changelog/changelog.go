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

package changelog

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Kind represents the type of a change.
type Kind int

const (
	kindChange = iota
	kindFeature
	kindEnhancement
	kindBugfix
)

func (k Kind) String() string {
	switch k {
	case kindChange:
		return "CHANGE"
	case kindFeature:
		return "FEATURE"
	case kindEnhancement:
		return "ENHANCEMENT"
	case kindBugfix:
		return "BUGFIX"
	}
	return ""
}

// Kinds is a list of Kind which implements sort.Interface.
type Kinds []Kind

func (k Kinds) Len() int           { return len(k) }
func (k Kinds) Less(i, j int) bool { return k[i] < k[j] }
func (k Kinds) Swap(i, j int)      { k[i], k[j] = k[j], k[i] }

// ParseKinds converts a slash-separated list of Kind to a list of Kind.
func ParseKinds(s string) Kinds {
	m := make(map[Kind]struct{})
	for _, k := range strings.Split(s, "/") {
		switch k {
		case "CHANGE":
			m[kindChange] = struct{}{}
		case "FEATURE":
			m[kindFeature] = struct{}{}
		case "ENHANCEMENT":
			m[kindEnhancement] = struct{}{}
		case "BUGFIX":
			m[kindBugfix] = struct{}{}
		}
	}

	var kinds Kinds
	for k := range m {
		kinds = append(kinds, k)
	}
	sort.Stable(kinds)
	return kinds
}

func (k Kinds) String() string {
	var s []string
	for i := range k {
		s = append(s, k[i].String())
	}
	return strings.Join(s, "/")
}

// Change represents a change description.
type Change struct {
	Text  string
	Kinds Kinds
}

type Changes []Change

func (c Changes) Sorted() error {
	less := func(k1, k2 Kinds) bool {
		if len(k1) == 0 {
			return len(k2) == 0
		}
		if len(k2) == 0 {
			return true
		}

		n := len(k1)
		if len(k1) > len(k2) {
			n = len(k2)
		}
		for j := 0; j < n; j++ {
			if k1[j] == k2[j] {
				continue
			}
			return k1[j] < k2[j]
		}
		return len(k1) <= len(k2)
	}

	for i := 0; i < len(c)-1; i++ {
		k1, k2 := c[i].Kinds, c[i+1].Kinds
		if !less(k1, k2) {
			return fmt.Errorf("%q should be after %q", c[i].Text, c[i+1].Text)
		}
	}
	return nil
}

// Entry represents an entry in the changelog.
type Entry struct {
	Version string
	Date    time.Time
	Changes Changes
	Text    string
}

const dateFormat = "2006-01-02"

// Name returns the canonical name of the entry.
func (c Entry) Name() string {
	return fmt.Sprintf("%s / %s", c.Version, c.Date.Format(dateFormat))
}

// ReadEntry reads the entry for the given version from the changelog file.
// It returns an error if the version is not found.
func ReadEntry(r io.Reader, version string) (*Entry, error) {
	reHeader, err := regexp.Compile(fmt.Sprintf(`^#{1,2} %s / (\d{4}-\d{2}-\d{2})`, regexp.QuoteMeta(version)))
	if err != nil {
		return nil, err
	}
	reChange := regexp.MustCompile(`^\* \[([^\]]+)\]`)

	var (
		reading bool
		lines   []string

		entry   = Entry{Version: version}
		scanner = bufio.NewScanner(r)
	)
	for (len(lines) == 0 || reading) && scanner.Scan() {
		line := scanner.Text()
		m := reHeader.FindStringSubmatch(line)
		switch {
		case len(m) > 0:
			reading = true
			t, err := time.Parse(dateFormat, m[1])
			if err != nil {
				return nil, fmt.Errorf("invalid changelog date: %w", err)
			}
			entry.Date = t
		case strings.HasPrefix(line, "## "):
			reading = false
		case reading:
			if len(lines) == 0 && strings.TrimSpace(line) == "" {
				continue
			}
			m := reChange.FindStringSubmatch(line)
			if len(m) > 1 {
				entry.Changes = append(entry.Changes, Change{Text: line, Kinds: ParseKinds(m[1])})
			}
			lines = append(lines, line)
		}
	}

	if entry.Date.IsZero() {
		return nil, fmt.Errorf(
			"unable to locate release information in changelog for version %q, expected format: %q",
			version,
			reHeader)
	}

	entry.Text = strings.Join(lines, "\n")
	return &entry, nil
}
