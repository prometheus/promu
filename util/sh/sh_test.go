// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sh

import (
	"reflect"
	"testing"
)

func TestSplitParameters(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "empty",
			in:   "",
			want: nil,
		},
		{
			name: "blanks only",
			in:   "  \t ",
			want: nil,
		},
		{
			name: "unquoted",
			in:   "-a -tags netgo",
			want: []string{"-a", "-tags", "netgo"},
		},
		{
			name: "extra blanks",
			in:   "  -a \t -mod=vendor  ",
			want: []string{"-a", "-mod=vendor"},
		},
		{
			name: "single quotes",
			in:   `-a -tags 'netgo static_build'`,
			want: []string{"-a", "-tags", "netgo static_build"},
		},
		{
			name: "double quotes",
			in:   `-a -tags "netgo static_build"`,
			want: []string{"-a", "-tags", "netgo static_build"},
		},
		{
			name: "quotes attached to the flag",
			in:   `-gcflags="all=-N -l"`,
			want: []string{"-gcflags=all=-N -l"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitParameters(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SplitParameters(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
