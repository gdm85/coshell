all: bin/coshell test

bin/coshell:
	mkdir -p bin/
	CGO_ENABLED=0 GOBIN="$(CURDIR)/bin/" go install
	strip bin/coshell

test:
	cd cosh && go test

clean:
	rm -rf bin/

fmt:
	gofmt -w *.go cosh/*.go

.PHONY: all bin/coshell test clean fmt
