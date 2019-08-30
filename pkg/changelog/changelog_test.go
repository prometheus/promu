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

package changelog

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestReadEntry(t *testing.T) {
	mustParse := func(s string) time.Time {
		d, err := time.Parse(dateFormat, s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return d
	}

	for _, tc := range []struct {
		in      string
		version string

		exp Entry
		err bool
	}{
		{
			// Version not found.
			in: `## 1.0.0 / 2016-01-02

* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
			version: "1.0.0-notfound",

			err: true,
		},
		{
			in: `## 1.0.0 / 2016-01-02

* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
			version: "1.0.0",

			exp: Entry{
				Version: "1.0.0",
				Date:    mustParse("2016-01-02"),
				Changes: []Change{
					{
						Kinds: []Kind{kindBugfix},
						Text:  "* [BUGFIX] Some fix.",
					},
					{
						Kinds: []Kind{kindFeature},
						Text:  "* [FEATURE] Some feature.",
					},
				},
				Text: `* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
			},
		},
		{
			in: `# 1.0.0 / 2016-01-02

* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
			version: "1.0.0",

			exp: Entry{
				Version: "1.0.0",
				Date:    mustParse("2016-01-02"),
				Changes: []Change{
					{
						Kinds: []Kind{kindBugfix},
						Text:  "* [BUGFIX] Some fix.",
					},
					{
						Kinds: []Kind{kindFeature},
						Text:  "* [FEATURE] Some feature.",
					},
				},
				Text: `* [BUGFIX] Some fix.
* [FEATURE] Some feature.`,
			},
		},
		{
			in: `## 1.0.0 / 2016-01-02
		
* [BUGFIX] Some fix.
* [FEATURE] Some feature.

## 0.0.1 / 2016-01-02

* [BUGFIX] Another fix.`,
			version: "1.0.0",

			exp: Entry{
				Version: "1.0.0",
				Date:    mustParse("2016-01-02"),
				Changes: []Change{
					{
						Kinds: []Kind{kindBugfix},
						Text:  "* [BUGFIX] Some fix.",
					},
					{
						Kinds: []Kind{kindFeature},
						Text:  "* [FEATURE] Some feature.",
					},
				},
				Text: `* [BUGFIX] Some fix.
* [FEATURE] Some feature.
`,
			},
		},
		{
			in: `## 1.0.0 / 2016-01-02

[BUGFIX] Some fix.
[FEATURE] Some feature.

## 0.0.1 / 2016-01-01

* [BUGFIX] Another fix.`,
			version: "0.0.1",

			exp: Entry{
				Version: "0.0.1",
				Date:    mustParse("2016-01-01"),
				Changes: []Change{
					{
						Kinds: []Kind{kindBugfix},
						Text:  "* [BUGFIX] Another fix.",
					},
				},
				Text: `* [BUGFIX] Another fix.`,
			},
		},
		{
			in: `## 1.0.0 / 2016-01-03
This is the first stable release.

* [CHANGE/BUGFIX] Some fix.
* [FEATURE] Some feature.

## 0.0.1 / 2016-01-02

* [BUGFIX] Another fix.`,
			version: "1.0.0",

			exp: Entry{
				Version: "1.0.0",
				Date:    mustParse("2016-01-03"),
				Changes: []Change{
					{
						Kinds: []Kind{kindChange, kindBugfix},
						Text:  "* [CHANGE/BUGFIX] Some fix.",
					},
					{
						Kinds: []Kind{kindFeature},
						Text:  "* [FEATURE] Some feature.",
					},
				},
				Text: `This is the first stable release.

* [CHANGE/BUGFIX] Some fix.
* [FEATURE] Some feature.
`,
			},
		},
		{
			in: `## 1.0.0 / 2016-01-04

### Breaking changes!

* [CHANGE] Some change.
* [FEATURE] Some feature.
* [ENHANCEMENT] Some enhancement.

## 0.0.1 / 2016-01-02

* [BUGFIX] Another fix.`,
			version: "1.0.0",

			exp: Entry{
				Version: "1.0.0",
				Date:    mustParse("2016-01-04"),
				Changes: []Change{
					{
						Kinds: []Kind{kindChange},
						Text:  "* [CHANGE] Some change.",
					},
					{
						Kinds: []Kind{kindFeature},
						Text:  "* [FEATURE] Some feature.",
					},
					{
						Kinds: []Kind{kindEnhancement},
						Text:  "* [ENHANCEMENT] Some enhancement.",
					},
				},
				Text: `### Breaking changes!

* [CHANGE] Some change.
* [FEATURE] Some feature.
* [ENHANCEMENT] Some enhancement.
`,
			},
		},
		{
			// Invalid date.
			in:      "## 1.0.0 / 2006-19-02",
			version: "1.0.0",

			err: true,
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			got, err := ReadEntry(bytes.NewBufferString(tc.in), tc.version)
			if tc.err {
				if err == nil {
					t.Fatal("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(&tc.exp, got) {
				t.Fatalf("expected:\n%v\ngot:\n%v", tc.exp, got)
			}
		})
	}
}

func TestKinds(t *testing.T) {
	for _, tc := range []struct {
		in  string
		exp Kinds
	}{
		{
			in:  "CHANGE",
			exp: Kinds{kindChange},
		},
		{
			in:  "BUGFIX/CHANGE",
			exp: Kinds{kindChange, kindBugfix},
		},
		{
			in:  "BUGFIX/BUGFIX",
			exp: Kinds{kindBugfix},
		},
		{
			in:  "BUGFIX/INVALID",
			exp: Kinds{kindBugfix},
		},
		{
			in: "INVALID",
		},
	} {
		t.Run("", func(t *testing.T) {
			got := ParseKinds(tc.in)
			if !reflect.DeepEqual(&tc.exp, &got) {
				t.Fatalf("expected:\n%v\ngot:\n%v", tc.exp, got)
			}
		})
	}
}

func TestChangesSorted(t *testing.T) {
	for _, tc := range []struct {
		in Changes

		err bool
	}{
		{
			in: Changes{
				{
					Kinds: Kinds{},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange},
				},
				{
					Kinds: Kinds{},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{},
				},
				{
					Kinds: Kinds{kindChange},
				},
				{
					Kinds: Kinds{kindFeature},
				},
			},
			err: true,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange},
				},
				{
					Kinds: Kinds{kindChange},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange},
				},
				{
					Kinds: Kinds{kindBugfix},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange},
				},
				{
					Kinds: Kinds{kindBugfix},
				},
				{
					Kinds: Kinds{kindFeature},
				},
			},
			err: true,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange},
				},
				{
					Kinds: Kinds{kindChange, kindBugfix},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange, kindBugfix},
				},
				{
					Kinds: Kinds{kindChange},
				},
			},
			err: true,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange, kindFeature, kindBugfix},
				},
				{
					Kinds: Kinds{kindChange, kindBugfix},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange, kindBugfix},
				},
				{
					Kinds: Kinds{kindChange, kindFeature, kindBugfix},
				},
			},
			err: true,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange, kindFeature, kindBugfix},
				},
				{
					Kinds: Kinds{kindChange, kindBugfix},
				},
				{
					Kinds: Kinds{},
				},
				{
					Kinds: Kinds{},
				},
			},
			err: false,
		},
		{
			in: Changes{
				{
					Kinds: Kinds{kindChange, kindFeature, kindBugfix},
				},
				{
					Kinds: Kinds{},
				},
				{
					Kinds: Kinds{kindChange, kindBugfix},
				},
				{
					Kinds: Kinds{},
				},
			},
			err: true,
		},
	} {
		t.Run("", func(t *testing.T) {
			err := tc.in.Sorted()
			if tc.err {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}
