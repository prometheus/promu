// Copyright 2019 The Prometheus Authors
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
	"bytes"
	"io/ioutil"
	"testing"
)

func TestIsPrerelease(t *testing.T) {
	for _, tc := range []struct {
		version string

		exp bool
		err bool
	}{
		{
			version: "1.0.0",
			exp:     false,
		},
		{
			version: "1.0.0-rc0",
			exp:     true,
		},
		{
			version: "x1.0.0-rc0",
			err:     true,
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			got, err := isPrerelease(tc.version)
			if err != nil && !tc.err {
				t.Fatalf("expected no error, got %v", err)
				return
			}
			if got != tc.exp {
				t.Fatalf("expected %t, got %t", tc.exp, got)
			}
		})
	}

}

func TestGetChangelog(t *testing.T) {
	for _, tc := range []struct {
		in      string
		version string

		header string
		body   string
	}{
		{
			in:      "",
			version: "1.0.0-notfound",
			header:  "",
			body:    "",
		},
		{
			in: `## 1.0.0 / 2016-01-02

* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
			version: "1.0.0",
			header:  "1.0.0 / 2016-01-02",
			body: `* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
		},
		{
			in: `## 1.0.0 / 2016-01-02

* [BUGFIX] Some fix.
* [FEATURE] Some feature.

## 0.0.1 / 2016-01-02

* [BUGFIX] Another fix.`,
			version: "1.0.0",
			header:  "1.0.0 / 2016-01-02",
			body: `* [BUGFIX] Some fix.
* [FEATURE] Some feature.
`,
		},
		{
			in: `## 1.0.0 / 2016-01-02

* [BUGFIX] Some fix.
* [FEATURE] Some feature.

## 0.0.1 / 2016-01-01

* [BUGFIX] Another fix.`,
			version: "0.0.1",
			header:  "0.0.1 / 2016-01-01",
			body:    `* [BUGFIX] Another fix.`,
		},
		{
			in: `## 1.0.0 / 2016-01-02
This is the first stable release.

* [BUGFIX] Some fix.
* [FEATURE] Some feature.

## 0.0.1 / 2016-01-02

* [BUGFIX] Another fix.`,
			version: "1.0.0",
			header:  "1.0.0 / 2016-01-02",
			body: `This is the first stable release.

* [BUGFIX] Some fix.
* [FEATURE] Some feature.
`,
		},
		{
			in: `## 1.0.0 / 2016-01-02

### Breaking changes!

* [BUGFIX] Some fix.
* [FEATURE] Some feature.

## 0.0.1 / 2016-01-02

* [BUGFIX] Another fix.`,
			version: "1.0.0",
			header:  "1.0.0 / 2016-01-02",
			body: `### Breaking changes!

* [BUGFIX] Some fix.
* [FEATURE] Some feature.
`,
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			header, body, err := getChangelog(tc.version, ioutil.NopCloser(bytes.NewBufferString(tc.in)))
			if err != nil && tc.version != "1.0.0-notfound" {
				t.Fatal(err)
			}
			if body != tc.body {
				t.Fatalf("expected body %q, got %q", tc.body, body)
			}
			if header != tc.header {
				t.Fatalf("expected header %q, got %q", tc.header, header)
			}
		})
	}
}
