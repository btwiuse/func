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

STAGE    ?= prod

ISSUER    = TODO
CLIENT_ID = TODO
ENDPOINT  = https://api.func.io/
ifeq ($(STAGE), dev)
	ISSUER    = https://dev-func.eu.auth0.com/
	CLIENT_ID = WiKX7zTA5lNbIPsx8HonmZS6IuldcyI6
	ENDPOINT  = https://dev-api.func.io/
endif

LDFLAGS  = -X main.Version=$(VERSION)
LDFLAGS += -X main.BuildDate=$(DATE)
LDFLAGS += -X main.DefaultIssuer=$(ISSUER)
LDFLAGS += -X main.DefaultEndpoint=$(ENDPOINT)
LDFLAGS += -X main.DefaultClientID=$(CLIENT_ID)
LDFLAGS += -s
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
