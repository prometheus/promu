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

package cmd

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCalculateSHA256s(t *testing.T) {
	dir, err := os.MkdirTemp("", "promu")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		filename = "testfile"
		location = filepath.Join(dir, filename)
		content  = []byte("temporary file's content")
		checksum = sha256.Sum256(content)
	)
	if err = os.WriteFile(location, content, 0666); err != nil {
		t.Fatal(err)
	}

	got, err := calculateSHA256s(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []checksumSHA256{
		{
			filename: filename,
			checksum: checksum[:],
		},
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want checksums %+v, got %+v", want, got)
	}
}
