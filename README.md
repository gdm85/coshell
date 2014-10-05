coshell v0.1.0
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

Build it with:

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

**NOTE:** order is not deterministic!
