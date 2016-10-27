all: bin/coshell test

bin/coshell:
	mkdir -p bin/
	GOBIN="$(CURDIR)/bin/" go install

test:
	cd cosh && go test

clean:
	rm -rf bin/

.PHONY: all bin/coshell test clean
