// Copyright © 2017 Prometheus Team
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
	"testing"
)

func TestToSemver(t *testing.T) {
	for _, tc := range []struct {
		version string

		exp string
	}{
		{
			version: "1.0.0",
			exp:     "1.0.0",
		},
		{
			version: "1.0.0-rc0",
			exp:     "1.0.0-rc0",
		},
		{
			version: "x1.0.0-rc0",
		},
		{
			version: "v1.0.0-rc0",
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			got, err := Info{Version: tc.version}.ToSemver()
			if err != nil {
				if tc.exp != "" {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if got.String() != tc.exp {
				t.Fatalf("expected %q, got %q", tc.exp, got)
			}
		})
	}
}

func TestRepoLocation(t *testing.T) {
	repoTests := []struct {
		s        string // input
		expected string // expected result
	}{
		{"git@github.com:prometheus/promu.git", "github.com/prometheus/promu"},
		{"https://github.com/prometheus/promu.git", "github.com/prometheus/promu"},
		{"ssh://git@gitlab.fr:22443/prometheus/promu.git", "gitlab.fr/prometheus/promu"},
		{"https://sdurrheimer@gitlab.fr/prometheus/promu.git", "gitlab.fr/prometheus/promu"},
	}

	for _, tc := range repoTests {
		actual, err := repoLocation(tc.s)
		if err != nil {
			t.Errorf("repoLocation(%s): %+v", tc.s, err)
		}
		if actual != tc.expected {
			t.Errorf("repoLocation(%s): expected %s, got %s", tc.s, tc.expected, actual)
		}
	}
}
