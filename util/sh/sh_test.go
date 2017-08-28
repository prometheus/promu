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
