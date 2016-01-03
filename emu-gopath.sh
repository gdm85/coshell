#!/bin/bash
## emu-gopath.sh
## @author gdm85
##
## emulate GOPATH structure for a build/test operation
##
#

if [ ! $# -eq 2 ]; then
	echo "Usage:"
	echo -e "\temu-gopath.sh example.com/author/package \"go test\"" 1>&2
	echo 1>&2
	echo "Must be run in workspace directory" 1>&2
	exit 1
fi

PKG="$1"
CMD="$2"

set -e

mkdir -p ".gopath/src/`dirname "$PKG"`"

if [ -L ".gopath/src/$PKG" ]; then
	unlink ".gopath/src/$PKG"
fi

ln -sf "$PWD" ".gopath/src/$PKG"

trap 'rm -rf .gopath' EXIT

GOPATH="$PWD/.gopath" sh -c "$CMD"
