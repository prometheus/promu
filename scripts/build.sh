#!/usr/bin/env bash

# Copyright 2015 The Prometheus Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eo pipefail

repo_path="github.com/prometheus/promu"

prefix=${1:-$(pwd)}
version=$( cat VERSION )
revision=$( git rev-parse --short HEAD 2> /dev/null || echo 'unknown' )
branch=$( git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown' )
host=$( hostname )
build_date=$( date +%Y%m%d-%H:%M:%S )
ext=""

if [ "$(go env GOOS)" = "windows" ]; then
  ext=".exe"
fi

ldflags="
  -X ${repo_path}/vendor/github.com/prometheus/common/version.Version=${version}
  -X ${repo_path}/vendor/github.com/prometheus/common/version.Revision=${revision}
  -X ${repo_path}/vendor/github.com/prometheus/common/version.Branch=${branch}
  -X ${repo_path}/vendor/github.com/prometheus/common/version.BuildUser=${USER}@${host}
  -X ${repo_path}/vendor/github.com/prometheus/common/version.BuildDate=${build_date}"

if [ "$(go env GOOS)" != "darwin"  ]; then
  ldflags="${ldflags} -extldflags \"-static\""
fi

export GO15VENDOREXPERIMENT="1"

echo " >   promu${ext}"
go build -ldflags "${ldflags}" -o "${prefix}/promu${ext}" ${repo_path}

exit 0
