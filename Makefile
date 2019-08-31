all: bin/coshell

bin/coshell:
	mkdir -p bin/
	CGO_ENABLED=0 go build -o bin/coshell .
	strip bin/coshell

test:
	cd cosh && go test

clean:
	rm -rf bin/

fmt:
	gofmt -w *.go cosh/*.go

.PHONY: all bin/coshell test clean fmt
