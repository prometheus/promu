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

	"github.com/pkg/errors"
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

// Kinds is a list of Kind.
type Kinds []Kind

// ParseKinds converts a slice of strings to a slice of Kind.
func ParseKinds(s []string) Kinds {
	m := make(map[Kind]struct{})
	for _, k := range s {
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
	sort.SliceStable(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

func (k Kinds) String() string {
	var s []string
	for i := range k {
		s = append(s, k[i].String())
	}
	return strings.Join(s, "/")
}

// Before returns whether the receiver should sort before the other.
func (k Kinds) Before(other Kinds) bool {
	if len(other) == 0 {
		return true
	} else if len(k) == 0 {
		return false
	}

	n := len(k)
	if len(k) > len(other) {
		n = len(other)
	}
	for j := 0; j < n; j++ {
		if k[j] == other[j] {
			continue
		}
		return k[j] < other[j]
	}
	return len(k) <= len(other)
}

// Change represents a change description.
type Change struct {
	Text  string
	Kinds Kinds
}

type Changes []Change

func (c Changes) Sorted() error {
	for i := 0; i < len(c)-1; i++ {
		if !c[i].Kinds.Before(c[i+1].Kinds) {
			return errors.Errorf("%q should be after %q", c[i].Text, c[i+1].Text)
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
				return nil, errors.Wrap(err, "invalid changelog date")
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
				kinds := strings.Split(m[1], "/")
				entry.Changes = append(entry.Changes, Change{Text: line, Kinds: ParseKinds(kinds)})
			}
			lines = append(lines, line)
		}
	}

	if entry.Date.IsZero() {
		return nil, errors.Errorf(
			"unable to locate release information in changelog for version %q, expected format: %q",
			version,
			reHeader)
	}

	entry.Text = strings.Join(lines, "\n")
	return &entry, nil
}
