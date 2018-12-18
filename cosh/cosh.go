/*
 * coshell v0.2.1 - a no-frills dependency-free replacement for GNU parallel
 * Copyright (C) 2014-2018 gdm85 - https://github.com/gdm85/coshell/

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

package cosh

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

const (
	OutputStdout = iota
	OutputStderr
)

var ErrEmptyCommandLine = errors.New("empty command line")

type OutputType int

type SortedOutput struct {
	sync.Mutex
	stdout SortedOutputWriter
	stderr SortedOutputWriter

	buffer bytes.Buffer

	segments []segment
}

type SortedOutputWriter struct {
	outputType OutputType
	parent     *SortedOutput
}

type segment struct {
	outputType OutputType
	offset     int
	length     int
}

func NewSortedOutput() *SortedOutput {
	sp := SortedOutput{}
	sp.stdout = SortedOutputWriter{OutputStdout, &sp}
	sp.stderr = SortedOutputWriter{OutputStderr, &sp}
	return &sp
}

func (so *SortedOutput) ReplayOutputs() error {
	so.Lock()
	defer so.Unlock()
	data := so.buffer.Bytes()

	for _, segment := range so.segments {
		if segment.outputType == OutputStdout {
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

func (sow SortedOutputWriter) Write(p []byte) (n int, err error) {
	sow.parent.Lock()

	offset := sow.parent.buffer.Len()
	n, err = sow.parent.buffer.Write(p)
	sow.parent.segments = append(sow.parent.segments, segment{sow.outputType, offset, n})

	sow.parent.Unlock()

	return
}

type CommandGroup struct {
	commands    []*exec.Cmd
	outputs     []*SortedOutput
	deinterlace bool
	halt        bool
	masterID    int
	ordered     bool

	shellArgs []string
}

func NewCommandGroup(shellArgs []string, deinterlace, halt bool, masterID int, ordered bool) *CommandGroup {
	if ordered {
		deinterlace = true
	}
	return &CommandGroup{
		deinterlace: deinterlace,
		halt:        halt,
		masterID:    masterID,
		ordered:     ordered,
	}
}

func (cg *CommandGroup) Start(jobs int) error {
	// run them all at once
	for i := 0; i < len(cg.commands); i++ {
		err := cg.commands[i].Start()
		if err != nil {
			return err
		}
	}

	return nil
}

type event struct {
	index    int
	error    error
	exitCode int
}

// Returns sum of all exit codes of individual commands
func (cg *CommandGroup) Join() (err error, exitCode int) {

	completedCommands := make(chan event, len(cg.commands))

	// join all and update cumulative exit status
	for i := 0; i < len(cg.commands); i++ {
		go func(i int) {
			if err := cg.commands[i].Wait(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
						completedCommands <- event{index: i, exitCode: status.ExitStatus()}
						return
					} else {
						completedCommands <- event{index: i, error: errors.New("cannot read exit error status")}
						return
					}
				} else {
					completedCommands <- event{index: i, error: err}
					return
				}
			}

			// completed successfully
			completedCommands <- event{index: i, exitCode: 0}
		}(i)
	}

	orderedDisplay := make([]event, len(cg.commands))
	displayedSoFar := 0

	count := 0
	for ev := range completedCommands {
		// print deinterlaced output
		if cg.deinterlace {
			if cg.ordered {
				// store ordered
				orderedDisplay[ev.index] = ev

				if displayedSoFar == ev.index {
					if err = cg.outputs[ev.index].ReplayOutputs(); err != nil {
						return
					}
					displayedSoFar++
				}
			} else {
				if err = cg.outputs[ev.index].ReplayOutputs(); err != nil {
					return
				}
			}
		}

		// an unexpected error during wait and exit code processing
		if ev.error != nil {
			err = ev.error
			return
		}

		exitCode += ev.exitCode
		count++

		if count == len(cg.commands) {
			break
		}

		// make these objects no more usable
		cg.commands[ev.index] = nil
		if cg.deinterlace {
			cg.outputs[ev.index] = nil
		}

		if cg.halt && ev.exitCode != 0 {
			if cg.deinterlace {
				// dump outputs that are available
				for _, output := range cg.outputs {
					if output != nil {
						_ = output.ReplayOutputs()
					}
				}
			}

			// exit point
			exitCode = ev.exitCode
			return
		}

		// if master aborts, show its output
		if cg.masterID != -1 && cg.masterID == ev.index {
			if cg.deinterlace {
				// dump outputs that are available
				for _, output := range cg.outputs {
					if output != nil {
						_ = output.ReplayOutputs()
					}
				}
			}

			// exit point
			exitCode = ev.exitCode
			return
		}
	}

	return
}

func (cg *CommandGroup) Add(commandLines ...string) error {
	// some common values for all commands
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	env := os.Environ()

	// prepare commands to be executed
	commands := make([]*exec.Cmd, len(commandLines))
	var outputs []*SortedOutput
	if cg.deinterlace {
		outputs = make([]*SortedOutput, len(commandLines))
		for i, _ := range outputs {
			outputs[i] = NewSortedOutput()
		}
	}
	for i := 0; i < len(commandLines); i++ {
		commands[i], err = cg.prepareCommand(commandLines[i])
		if err != nil {
			// will only happen in case of problems at splting the command line
			return err
		}
		commands[i].Env = env
		commands[i].Dir = cwd

		if cg.deinterlace {
			commands[i].Stdout = outputs[i].stdout
			commands[i].Stderr = outputs[i].stderr
		} else {
			commands[i].Stdout = os.Stdout
			commands[i].Stderr = os.Stderr
			// notice here how no stdin is attached to commands
		}
	}

	// finally append
	cg.commands = append(cg.commands, commands...)
	if cg.deinterlace {
		cg.outputs = append(cg.outputs, outputs...)
	}

	return nil
}

func (cg *CommandGroup) prepareCommand(cmdLine string) (*exec.Cmd, error) {
	// using a shell prefix, append the whole command line
	if len(cg.shellArgs) != 0 {
		return exec.Command(cg.shellArgs[0], append(cg.shellArgs[1:], cmdLine)...), nil
	}

	args, err := Split(cmdLine)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, ErrEmptyCommandLine
	}

	return exec.Command(args[0], args[1:]...), nil
}
