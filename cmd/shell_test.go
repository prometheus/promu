package cmd

import (
	"strings"
	"testing"

	"github.com/prometheus/promu/util/sh"
)

func TestSplitParameters(t *testing.T) {
	in := `-a -tags 'netgo static_build'`
	expect := []string{"-a", "-tags", `'netgo static_build'`}
	got := sh.SplitParameters(in)
	for i, g := range got {
		if expect[i] != g {
			t.Error("expected", expect[i], "got", g, "full output: ", strings.Join(got, "#"))
		}
	}
}
