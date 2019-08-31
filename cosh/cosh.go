/*
 * coshell v0.2.2 - a no-frills dependency-free replacement for GNU parallel
 * Copyright (C) 2014-2019 gdm85 - https://github.com/gdm85/coshell/

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
	"fmt"
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

type event struct {
	index    int
	err      error
	exitCode int
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

	completedCommands chan event
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
		shellArgs:   shellArgs,
	}
}

func (cg *CommandGroup) Start(jobs int) error {
	if jobs == 0 {
		// set to maximum possible
		jobs = len(cg.commands)
	}
	res := make(chan struct{}, jobs)
	for i := 0; i < jobs; i++ {
		res <- struct{}{}
	}

	cg.completedCommands = make(chan event, len(cg.commands))

	// run them all at once
	for i := 0; i < len(cg.commands); i++ {
		go func(i int) {
			<-res
			defer func() {
				res <- struct{}{}
			}()

			err := cg.commands[i].Start()
			if err != nil {
				cg.completedCommands <- event{index: i, err: err}
				return
			}

			if err := cg.commands[i].Wait(); err == nil {
				// completed successfully
				cg.completedCommands <- event{index: i, exitCode: 0}
				return
			}
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					cg.completedCommands <- event{
						index:    i,
						exitCode: status.ExitStatus(),
					}
					return
				}
				cg.completedCommands <- event{index: i, err: errors.New("cannot read exit error status")}
				return
			}
			// any other error
			cg.completedCommands <- event{index: i, err: err}
			return
		}(i)
	}

	return nil
}

// Join waits for all commands to complete execution and return the sum of each individual exit code.
func (cg *CommandGroup) Join() (int, error) {

	orderedDisplay := make([]event, len(cg.commands))
	displayedSoFar := 0

	var outputErrors []error

	count := 0
	exitCode := 0
	for ev := range cg.completedCommands {
		// print deinterlaced output
		if cg.deinterlace {
			if cg.ordered {
				// store ordered
				orderedDisplay[ev.index] = ev

				if displayedSoFar == ev.index {
					if err := cg.outputs[ev.index].ReplayOutputs(); err != nil {
						outputErrors = append(outputErrors, err)
					}
					displayedSoFar++
				}
			} else {
				if err := cg.outputs[ev.index].ReplayOutputs(); err != nil {
					outputErrors = append(outputErrors, err)
				}
			}
		}

		// an unexpected error during wait and exit code processing
		if ev.err != nil {
			cg.terminateAll(ev.index)
			return -1, ev.err
		}

		exitCode += ev.exitCode
		count++

		// reached total commands that were started
		if count == len(cg.commands) {
			close(cg.completedCommands)
			cg.completedCommands = nil
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
						err := output.ReplayOutputs()
						if err != nil {
							outputErrors = append(outputErrors, err)
						}
					}
				}
			}

			// exit point
			exitCode = ev.exitCode
			cg.terminateAll(ev.index)
			return exitCode, nil
		}

		// if master aborts, show its output
		if cg.masterID != -1 && cg.masterID == ev.index {
			if cg.deinterlace {
				// dump outputs that are available
				for _, output := range cg.outputs {
					if output != nil {
						err := output.ReplayOutputs()
						if err != nil {
							outputErrors = append(outputErrors, err)
						}
					}
				}
			}

			// exit point
			exitCode = ev.exitCode
			cg.terminateAll(ev.index)
			return exitCode, nil
		}
	}

	if len(outputErrors) != 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %d errors while replaying output\n", len(outputErrors))
	}

	return exitCode, nil
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

func (cg *CommandGroup) terminateAll(exceptIndex int) {
	for i, cmd := range cg.commands {
		if cmd == nil {
			// already exited
			continue
		}
		if i == exceptIndex {
			continue
		}
		// not yet started processes
		if cmd.Process == nil {
			continue
		}

		err := cmd.Process.Kill()
		if err != nil && err.Error() != "os: process already finished" {
			fmt.Fprintf(os.Stderr, "ERROR: could not kill process %d: %v\n", cmd.Process.Pid, err)
		}
	}
}
