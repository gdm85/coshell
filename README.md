# coshell v0.2.4

A no-frills dependency-free replacement for GNU parallel, perfect for initramfs usage.

Licensed under GNU/GPL v2.

# How it works

An ``sh -c ...`` command is started for each of the input commands; environment and current working directory are preserved.
**NOTE:** file descriptors are not

All commands will be executed, no matter which one fails.
Return value will be the sum of exit values of each command.

It is suggested to use `exec` if you want the shell-spawned process to substitute each of the wrapping shells and be able to handle signals.
See also http://tldp.org/LDP/abs/html/process-sub.html
Alternatively, you can use `--shell=""` to force the usage of no shell (see description in the options section).

# Self-contained

    $ ldd coshell
    	not a dynamic executable

Thanks to [Go language](https://golang.org/), this is a self-contained executable thus a perfect match for inclusion in an initramfs or any other project where you would prefer not to have too many dependencies.

# Installation

Once you run:

    go get github.com/gdm85/coshell

The binary will be available in your ``$GOPATH/bin``; alternatively, build it with:

    make

Then copy the output ``bin/coshell`` binary to your `$PATH`, ``~/bin``, ``/usr/local/bin`` or any of your option.

# Usage

Specify each command on a single line as standard input.

Example:

    echo -e "echo test1\necho test2\necho test3" | coshell

Output:

    test3
    test1
    test2

## sequence length option

By specifying a sequence length greater than 1 it is possible to group commands in sequences. Each group of commands will be executed sequentially.

## deinterlace option

Order is not deterministic by default, but with option ``--deinterlace`` or ``-d`` all output will be buffered and afterwards
printed in the same chronological order as your input.

## shell

It is possible to specify a custom shell prefix or no shell at all (`--shell=""`); in such case, commands will be split
according to /bin/sh's word-splitting rules. It supports backslash-escapes, single-quotes, and double-quotes.
Notably it does not support the `$''` style of quoting. It also doesn't attempt to perform any other sort of
expansion, including brace expansion, shell expansion, or pathname expansion.

If the given input has an unterminated quoted string or ends in a backslash-escape an error is returned.

## halt-all option

If `--halt-all` or `-a` option is specified then first process to terminate unsuccessfully (with non-zero exit code) will cause 
all processes to immediately exit (including coshell) with the exit code of such process.

## master option

The `--master=n` or `-m=n` option takes a positive integer number as the index of specified command lines to identify
which process "leads" the pack: when the process exits all neighbour processes will be terminated as well and its exit code
will be adopted as coshell exit code.

## Examples

See [examples/](examples/) directory for examples of various use-cases.
