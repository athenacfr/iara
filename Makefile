BINARY      := cw
UNAME_S     := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
INSTALL_DIR ?= /usr/local/bin
else
INSTALL_DIR ?= $(HOME)/.local/bin
endif
INSTALL     := $(INSTALL_DIR)/$(BINARY)

.PHONY: build install clean test e2e e2e-update test-tui

build:
	go build -o bin/$(BINARY) .

install: build
	rm -f $(INSTALL)
	cp bin/$(BINARY) $(INSTALL)

clean:
	rm -f bin/$(BINARY)

test:
	go test ./...

e2e: build
	cd tests && bun install --frozen-lockfile 2>/dev/null || cd tests && bun install
	cd tests && rm -rf .tui-test/cache
	cd tests && CW_TEST_ROOT=$(CURDIR) bun run test

e2e-update: build
	cd tests && bun install --frozen-lockfile 2>/dev/null || cd tests && bun install
	cd tests && rm -rf .tui-test/cache
	cd tests && CW_TEST_ROOT=$(CURDIR) npx tui-test --updateSnapshot

test-tui: e2e
