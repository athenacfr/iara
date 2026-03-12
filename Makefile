BINARY  := cw
INSTALL := $(HOME)/.local/bin/$(BINARY)

.PHONY: build install clean

build:
	go build -o bin/$(BINARY) .

install: build
	rm -f $(INSTALL)
	cp bin/$(BINARY) $(INSTALL)

clean:
	rm -f bin/$(BINARY)
