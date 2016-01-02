/*
 * coshell v0.1.4 - a no-frills dependency-free replacement for GNU parallel
 * Copyright (C) 2014-2015 gdm85 - https://github.com/gdm85/coshell/

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
	"strconv"
	"strings"

	"github.com/gdm85/coshell/cosh"
)

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "coshell: %s\n", err)
	os.Exit(1)
}

func main() {
	var deinterlace, halt, nextMightBeMasterId bool
	masterId := -1
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			if nextMightBeMasterId {
				nextMightBeMasterId = false
				if len(os.Args[i]) == 0 {
					fmt.Fprintf(os.Stderr, "Empty master command index specified\n")
					os.Exit(1)
					return
				}

				// clearly not a number
				if os.Args[i][0] == '-' {
					// use default
					masterId = 0
					continue
				}

				// argument is not starting with dash, number expected
				i, err := strconv.Atoi(os.Args[i])
				if err != nil || i < 0 {
					fmt.Fprintf(os.Stderr, "Invalid master command index specified: %s\n", os.Args[i])
					os.Exit(1)
					return
				}
				masterId = i
				continue
			}

			// check parameter of --master option
			var remainder string
			var found bool
			if strings.HasPrefix(os.Args[i], "--master") {
				remainder = os.Args[i][len("--master"):]
				found = true
			} else if strings.HasPrefix(os.Args[i], "-m") {
				remainder = os.Args[i][len("-m"):]
				found = true
			}
			if found {
				if len(remainder) == 0 {
					nextMightBeMasterId = true
					continue
				}
				if remainder[0] == '=' {
					remainder = remainder[1:]
				}
				i, err := strconv.Atoi(remainder)
				if err != nil || i < 0 {
					fmt.Fprintf(os.Stderr, "Invalid master command index specified: %s\n", remainder)
					os.Exit(1)
					return
				}
				masterId = i
				continue
			}

			switch os.Args[i] {
			case "--help", "-h":
				fmt.Printf("coshell v0.1.4 by gdm85 - Licensed under GNU GPLv2\n")
				fmt.Printf("Usage:\n\tcoshell [--help|-h] [--deinterlace|-d] [--halt-all|-a] < list-of-commands\n")
				fmt.Printf("\t\t--deinterlace | -d\t\tShow individual output of processes in blocks, second order of termination\n\n")
				fmt.Printf("\t\t--halt-all | -a\t\tTerminate neighbour processes as soon as any has failed, using its exit code\n\n")
				fmt.Printf("\t\t--master=0 | -m=0\t\tTerminate neighbour processes as soon as command from specified input line exits and use its exit code; if no id is specified, 0 is assumed\n\n")
				fmt.Printf("Each line read from standard input will be run as a command via `sh -c`\n")
				os.Exit(0)
			case "--halt-all", "-a":
				halt = true
				continue
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
			return
		}

		commandLines = append(commandLines, line)
	}

	if len(commandLines) == 0 {
		fatal(errors.New("please specify at least 1 command in standard input"))
		return
	}
	if masterId != -1 && masterId >= len(commandLines) {
		fatal(errors.New("specified master command index is beyond last specified command"))
		return
	}

	cg := cosh.NewCommandGroup(deinterlace, halt, masterId)

	err := cg.Add(commandLines...)
	if err != nil {
		fatal(err)
		return
	}

	err = cg.Start()
	if err != nil {
		fatal(err)
		return
	}

	err, exitCode := cg.Join()
	if err != nil {
		fatal(err)
		return
	}

	os.Exit(exitCode)
}
