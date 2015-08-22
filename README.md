coshell v0.1.1
==============

A no-frills dependency-free replacement for GNU parallel, perfect for initramfs usage.

Licensed under GNU/GPL v2.

How it works
============

An ``sh -c ...`` command is started for each of the input commands; environment and current working directory are preserved.
**NOTE:** file descriptors are not

All commands will be executed, no matter which one fails.
Return value will be the sum of exit values of each command.

Self-contained
==============

    $ ldd coshell
    	not a dynamic executable

Thanks to [Go language](https://golang.org/), this is a self-contained executable thus a perfect match for inclusion in an initramfs.

Installation
============

Once you run:

    go get github.com/gdm85/coshell

The binary will be available in your ``$GOPATH/bin``; alternatively, build it with:

    go build

Then copy the ``coshell`` binary to your PATH, ``~/bin``, ``/usr/local/bin`` or any of your option.

Usage
=====

Specify each command on a single line as standard input.

Example:

    echo -e "echo test1\necho test2\necho test3" | coshell

Output:

    test3
    test1
    test2

Order is not deterministic by default, but if you specify ``--deinterlace`` option all output will be buffered and afterwards
printed in the original order of specified commands.

See also examples in ``examples/`` directory.
