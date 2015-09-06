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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

const (
	outputStdout = iota
	outputStderr
)

type outputType int

type sortedOutput struct {
	sync.Mutex
	stdout sortedOutputWriter
	stderr sortedOutputWriter

	buffer bytes.Buffer

	segments []segment
}

type sortedOutputWriter struct {
	outputType outputType
	parent     *sortedOutput
}

type segment struct {
	outputType outputType
	offset     int
	length     int
}

var (
	deinterlace = false
)

func NewSortedOutput() *sortedOutput {
	sp := sortedOutput{}
	sp.stdout = sortedOutputWriter{outputStdout, &sp}
	sp.stderr = sortedOutputWriter{outputStderr, &sp}
	return &sp
}

func (so *sortedOutput) ReplayOutputs() error {
	so.Lock()
	defer so.Unlock()
	data := so.buffer.Bytes()

	for _, segment := range so.segments {
		if segment.outputType == outputStdout {
			if _, err := os.Stdout.Write(data[segment.offset : segment.offset+segment.length]); err != nil {
				return err
			}
			continue
		}
		// if it's not stdout, then it's stderr
		if _, err := os.Stderr.Write(data[segment.offset : segment.offset+segment.length]); err != nil {
			return err
		}
	}

	// reset
	so.segments = []segment{}
	so.buffer.Reset()

	return nil
}

func (sow sortedOutputWriter) Write(p []byte) (n int, err error) {
	sow.parent.Lock()

	offset := sow.parent.buffer.Len()
	n, err = sow.parent.buffer.Write(p)
	sow.parent.segments = append(sow.parent.segments, segment{sow.outputType, offset, n})

	sow.parent.Unlock()

	return
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "coshell: %s\n", err)
	os.Exit(1)
}

func main() {
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--help":
				fmt.Printf("coshell v0.1.1 by gdm85 - Licensed under GNU GPLv2\n")
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

	// some common values for all commands
	env := os.Environ()
	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	// prepare commands to be executed
	commands := make([]*exec.Cmd, len(commandLines))
	var outputs []*sortedOutput
	if deinterlace {
		outputs = make([]*sortedOutput, len(commandLines))
		for i, _ := range outputs {
			outputs[i] = NewSortedOutput()
		}
	}
	for i := 0; i < len(commands); i++ {
		commands[i] = exec.Command("sh", "-c", commandLines[i])
		commands[i].Env = env
		commands[i].Dir = cwd

		if deinterlace {
			commands[i].Stdout = outputs[i].stdout
			commands[i].Stderr = outputs[i].stderr
		} else {
			commands[i].Stdout = os.Stdout
			commands[i].Stderr = os.Stderr
			// notice here how no stdin is attached to commands
		}
	}

	// run them all at once
	for i := 0; i < len(commands); i++ {
		err := commands[i].Start()
		if err != nil {
			fatal(err)
		}
	}

	// sum of all exit codes of individual commands
	exitCode := 0

	// join all and update cumulative exit status
	for i := 0; i < len(commands); i++ {
		if err := commands[i].Wait(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode += status.ExitStatus()
				} else {
					fatal(errors.New("cannot read exit status"))
				}
			} else {
				fatal(err)
			}
		}

		// print deinterlaced output
		if deinterlace {
			if err := outputs[i].ReplayOutputs(); err != nil {
				fatal(err)
			}
		}
	}

	os.Exit(exitCode)
}
