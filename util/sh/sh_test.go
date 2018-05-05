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
	"strings"
	"testing"
)

func TestSplitParameters(t *testing.T) {
	in := `-a -tags 'netgo static_build'`
	expect := []string{"-a", "-tags", `'netgo static_build'`}
	got := SplitParameters(in)
	for i, g := range got {
		if expect[i] != g {
			t.Error("expected", expect[i], "got", g, "full output: ", strings.Join(got, "#"))
		}
	}
}
