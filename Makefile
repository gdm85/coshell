all: build

build:
	mkdir -p .gopath/src/github.com/gdm85/ bin/
	if [ -L .gopath/src/github.com/gdm85/coshell ]; then unlink .gopath/src/github.com/gdm85/coshell; fi
	ln -sf "$(CURDIR)" .gopath/src/github.com/gdm85/coshell
	GOPATH="$(CURDIR)/.gopath" GOBIN="bin/" go install
	rm -rf .gopath

clean:
	rm -rf bin/

.PHONY: all build clean
