/*
 * coshell v0.1.0 - a no-frills dependency-free replacement for GNU parallel
 * Copyright (C) 2014 gdm85 - https://github.com/gdm85/coshell/

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
*/

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	// show some basic help, when asked to
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--help":
				fmt.Fprintf(os.Stdout, "Usage:\n\tcoshell < list-of-commands\n\tEach line will be run as a command\n")
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "Invalid parameter specified: %s\n", os.Args[i])
			}
			os.Exit(1)
		}
	}

	// collect all commands to run from stdin
	var commandLines []string

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				break
			}

			// crash in case of other errors
			panic(err)
		}

		commandLines = append(commandLines, line)
	}

	if len(commandLines) == 0 {
		fmt.Fprintf(os.Stderr, "coshell: please specify at least 1 command\n")
		os.Exit(1)
	}

	// some common values for all commands
	env := os.Environ()
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// prepare commands to be executed
	commands := make([]*exec.Cmd, len(commandLines))
	for i := 0; i < len(commands); i++ {
		commands[i] = exec.Command("sh", "-c", commandLines[i])
		commands[i].Env = env
		commands[i].Dir = cwd
		commands[i].Stdout = os.Stdout
		commands[i].Stderr = os.Stderr
		// notice here how no stdin is attached to commands
	}

	// run them all at once
	for i := 0; i < len(commands); i++ {
		err := commands[i].Start()
		if err != nil {
			panic(err)
		}
	}

	// sum of all exit codes
	exitCode := 0

	// join all and update cumulative exit status
	for i := 0; i < len(commands); i++ {
		if err := commands[i].Wait(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode += status.ExitStatus()
				} else {
					panic("cannot read exit status")
				}
			} else {
				panic(err)
			}
		}
	}

	os.Exit(exitCode)
}
