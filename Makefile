#!/usr/bin/make -f

SHELL=/bin/bash -o pipefail

MODULE   = $(shell env GO111MODULE=on go list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null)
BIN      = ./bin

SRC      = $(shell find . -name *.go)
BINARIES = $(shell ls cmd)
TARGETS  = $(patsubst %, $(BIN)/%, $(BINARIES))
INSTALL  = $(patsubst %, $$GOPATH/bin/%, $(BINARIES))

LDFLAGS  = -ldflags "-X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.BuildDate=$(DATE)"

.PHONY: all
all: test $(TARGETS)

.PHONY: test
test:
	go test ./...

.PHONY: build
build: $(TARGETS)

.PHONY: install
install: $(INSTALL)

.PHONY: uninstall
uninstall:
	rm -f $(INSTALL)

.PHONY: clean
clean:
	@rm -rf $(BIN)

.PHONY: version
version:
	@echo $(VERSION)

$(BIN)/%: $(SRC)
	@mkdir -p $(dir $@)
	go build \
		$(LDFLAGS) \
		-o $(BIN)/$(notdir $(MODULE)) \
		./cmd/$*
	@du -h $@

$$GOPATH/bin/%:
	go install \
		$(LDFLAGS) \
		./cmd/$*
