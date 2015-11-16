/*
 * coshell v0.1.1 - a no-frills dependency-free replacement for GNU parallel
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
	"errors"
	"fmt"
	"io"
	"os"

	cosh "./cosh"
)

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "coshell: %s\n", err)
	os.Exit(1)
}

func main() {
	deinterlace := false
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--help":
				fmt.Printf("coshell v0.1.2 by gdm85 - Licensed under GNU GPLv2\n")
				fmt.Printf("Usage:\n\tcoshell [--deinterlace|-d] < list-of-commands\n")
				fmt.Printf("\t\t--deinterlace | -d\t\tRe-order stdout/stderr second original order of running programs\n\n")
				fmt.Printf("Each line read from standard input will be run as a command via `sh -c`\n")
				fmt.Printf("NOTE: when using --deinterlace, output will necessarily be buffered\n")
				os.Exit(0)
			case "--deinterlace", "-d":
				deinterlace = true
				continue
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
			fatal(err)
		}

		commandLines = append(commandLines, line)
	}

	if len(commandLines) == 0 {
		fatal(errors.New("please specify at least 1 command in standard input"))
	}

	cg := cosh.NewCommandGroup(deinterlace)

	err := cg.Add(commandLines...)
	if err != nil {
		fatal(err)
	}

	err = cg.Start()
	if err != nil {
		fatal(err)
	}

	err, exitCode := cg.Join()
	if err != nil {
		fatal(err)
	}

	os.Exit(exitCode)
}
