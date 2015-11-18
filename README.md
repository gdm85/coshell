# coshell v0.1.2

A no-frills dependency-free replacement for GNU parallel, perfect for initramfs usage.

Licensed under GNU/GPL v2.

# How it works

An ``sh -c ...`` command is started for each of the input commands; environment and current working directory are preserved.
**NOTE:** file descriptors are not

All commands will be executed, no matter which one fails.
Return value will be the sum of exit values of each command.

It is suggested to use `exec` if you want the shell-spawned process to subsitute the shell and be responsibe to signals.
See also http://tldp.org/LDP/abs/html/process-sub.html

# Self-contained

    $ ldd coshell
    	not a dynamic executable

Thanks to [Go language](https://golang.org/), this is a self-contained executable thus a perfect match for inclusion in an initramfs or any other project where you would prefer not to have too many dependencies.

# Installation

Once you run:

    go get github.com/gdm85/coshell

The binary will be available in your ``$GOPATH/bin``; alternatively, build it with:

    go build

Then copy the ``coshell`` binary to your PATH, ``~/bin``, ``/usr/local/bin`` or any of your option.

# Usage

Specify each command on a single line as standard input.

Example:

    echo -e "echo test1\necho test2\necho test3" | coshell

Output:

    test3
    test1
    test2

## deinterlace option

Order is not deterministic by default, but with option ``--deinterlace`` or ``-d`` all output will be buffered and afterwards
printed in the same chronological order as process termination.

## halt-all option

If `--halt-all` or `-a` option is specified then first process to terminate unsuccessfully (with non-zero exit code) will cause 
all processes to immediately exit (including coshell) with the exit code of such process.

## master option

The `--master=n` or `-m=n` option takes a positive integer number as the index of specified command lines to identify
which process "leads" the pack: when the process exits all neighbour processes will be terminated as well and its exit code
will be adopted as coshell exit code.

## Examples

See [examples/](examples/) directory.
