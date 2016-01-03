all: build test

build:
	mkdir -p bin/
	./emu-gopath.sh github.com/gdm85/coshell 'GOBIN="bin/" go install'

test:
	./emu-gopath.sh github.com/gdm85/coshell 'cd cosh && go test'

clean:
	rm -rf bin/

.PHONY: all build clean
