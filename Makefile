# Copyright © 2016 Prometheus Team
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO           ?= GO15VENDOREXPERIMENT=1 go
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
PROMU        ?= $(FIRST_GOPATH)/bin/promu
STATICCHECK  ?= $(FIRST_GOPATH)/bin/staticcheck
pkgs          = $(shell $(GO) list ./... | grep -v /vendor/)

PREFIX       ?= $(shell pwd)
BIN_DIR      ?= $(shell pwd)


all: format style vet staticcheck test build

build: $(PROMU)
	@echo ">> building binaries"
	@$(PROMU) build --prefix $(PREFIX)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

$(FIRST_GOPATH)/bin/promu promu:
	@GOOS=$(shell uname -s | tr A-Z a-z) \
		GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(patsubst arm%,arm,$(shell uname -m)))) \
		$(GO) install github.com/prometheus/promu

$(FIRST_GOPATH)/bin/staticcheck:
	@GOOS= GOARCH= $(GO) get -u honnef.co/go/tools/cmd/staticcheck

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

tarball: $(PROMU)
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

test:
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

staticcheck: $(STATICCHECK)
	@echo ">> running staticcheck"
	@$(STATICCHECK) $(pkgs)

.PHONY: all build format promu style tarball test vet staticcheck $(FIRST_GOPATH)/bin/promu $(FIRST_GOPATH)/bin/staticcheck
