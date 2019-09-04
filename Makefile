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

LDFLAGS  = -X main.Version=$(VERSION)
LDFLAGS += -X main.BuildDate=$(DATE)

# Auth flags
LDFLAGS += -X $(MODULE)/auth.Issuer=$(ISSUER)
LDFLAGS += -X $(MODULE)/auth.ClientID=$(CLIENT_ID)
LDFLAGS += -X $(MODULE)/auth.Audience=$(ENDPOINT)

# Omit the symbol table and debug information.
LDFLAGS += -s
# Omit the DWARF symbol table.
LDFLAGS += -w

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
		-ldflags "$(LDFLAGS)" \
		-o $@ \
		./cmd/$*
	@du -h $@

$$GOPATH/bin/%:
	go install \
		-ldflags "$(LDFLAGS)" \
		./cmd/$*
