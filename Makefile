BINARY      := cw
INSTALL_DIR ?= $(HOME)/.local/bin
INSTALL     := $(INSTALL_DIR)/$(BINARY)

.PHONY: build install clean

build:
	go build -o bin/$(BINARY) .

install: build
	rm -f $(INSTALL)
	cp bin/$(BINARY) $(INSTALL)

clean:
	rm -f bin/$(BINARY)
