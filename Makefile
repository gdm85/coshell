all: bin/coshell test

bin/coshell:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/coshell .
	strip bin/coshell

test:
	go test -race -v ./cosh

clean:
	rm -rf bin/

fmt:
	gofmt -w *.go cosh/*.go

.PHONY: all bin/coshell test clean fmt
