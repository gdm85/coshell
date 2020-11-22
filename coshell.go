/*
 * coshell v0.2.4 - a no-frills dependency-free replacement for GNU parallel
 * Copyright (C) 2014-2020 gdm85 - https://github.com/gdm85/coshell/

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
	"strings"

	"github.com/gdm85/coshell/cosh"

	flag "github.com/ogier/pflag"
)

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "coshell: %s\n", err)
	os.Exit(1)
}

func main() {
	var (
		version        bool
		cfg            = cosh.DefaultCommandPoolConfig
		jobs           int
		sequenceLength int
		shellArgs      string
	)

	flag.BoolVarP(&version, "version", "v", false, "Display version and exit")
	flag.BoolVarP(&cfg.Deinterlace, "deinterlace", "d", false, "Show individual output of processes in blocks, second order of termination")
	flag.BoolVarP(&cfg.Halt, "halt-all", "a", false, "Terminate neighbour processes as soon as any has failed, using its exit code")
	flag.IntVarP(&cfg.MasterID, "master", "m", -1, "Terminate neighbour processes as soon as command from specified input line exits and use its exit code; multiplied by sequence-length")
	flag.IntVarP(&sequenceLength, "sequence-length", "l", 1, "Execute this amount of lines in sequence; corresponds to '&&' shell command concatenation.")
	flag.IntVarP(&jobs, "jobs", "j", 8, "Use specified number of jobs; specify 0 for unlimited concurrency")
	flag.StringVarP(&shellArgs, "shell", "s", "sh -c", "If specified, the specified space-separated arguments will be used as shell prefix and the whole line will be passed as a single argument")

	showVersion := func() {
		fmt.Fprintf(os.Stderr, "coshell v0.2.4 by gdm85 - Licensed under GNU GPLv2\n")
	}

	flag.Usage = func() {
		showVersion()
		fmt.Fprintf(os.Stderr, "Usage:\n\tcoshell [--jobs=8|-j8] [--deinterlace|-d] [--halt-all|-a] < list-of-commands\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Each line read from standard input will be run as a command via `sh -c` (can be overriden with --shell=); empty lines are ignored\n")
	}

	flag.Parse()

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "Invalid arguments specified\n")
		os.Exit(1)
	}

	if version {
		showVersion()
		os.Exit(0)
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

		line = strings.TrimSuffix(line, "\n")

		if len(line) == 0 {
			continue
		}

		commandLines = append(commandLines, line)
	}

	if sequenceLength < 1 {
		fatal(errors.New("sequence length must be at least 1"))
	}

	if len(commandLines) == 0 {
		fatal(errors.New("please specify at least 1 command in standard input"))
		return
	}
	if cfg.MasterID != -1 && cfg.MasterID >= len(commandLines)/sequenceLength {
		fatal(errors.New("specified master command index is beyond last specified command"))
		return
	}

	if len(commandLines)%sequenceLength != 0 {
		fatal(errors.New("specified commands must be a multiple of sequence length"))
	}

	if jobs < 0 {
		fatal(errors.New("invalid jobs number"))
		return
	}

	if shellArgs != "" {
		cfg.ShellArgs = strings.Split(shellArgs, " ")
	}

	cg := cosh.NewCommandPool(&cfg)

	err := cg.Add(sequenceLength, commandLines...)
	if err != nil {
		fatal(err)
		return
	}

	err = cg.Start(jobs)
	if err != nil {
		fatal(err)
		return
	}

	exitCode, err := cg.Join()
	if err != nil {
		fatal(err)
		return
	}

	os.Exit(exitCode)
}
