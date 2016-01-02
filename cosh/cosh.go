/*
 * coshell v0.1.3 - a no-frills dependency-free replacement for GNU parallel
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
	masterId    int
}

func NewCommandGroup(deinterlace, halt bool, masterId int) *CommandGroup {
	return &CommandGroup{deinterlace: deinterlace, halt: halt, masterId: masterId}
}

func (cg *CommandGroup) Start() error {
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
	i        int
	error    error
	exitCode int
}

// Returns sum of all exit codes of individual commands
func (cg *CommandGroup) Join() (err error, exitCode int) {

	chainOfEvents := make(chan event, len(cg.commands))

	// join all and update cumulative exit status
	for i := 0; i < len(cg.commands); i++ {
		go func(i int) {
			if err := cg.commands[i].Wait(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
						chainOfEvents <- event{i: i, exitCode: status.ExitStatus()}
						return
					} else {
						chainOfEvents <- event{i: i, error: errors.New("cannot read exit error status")}
						return
					}
				} else {
					chainOfEvents <- event{i: i, error: err}
					return
				}
			}

			chainOfEvents <- event{i: i, exitCode: 0}
		}(i)
	}

	count := 0
	for ev := range chainOfEvents {
		// print deinterlaced output
		if cg.deinterlace {
			if err = cg.outputs[ev.i].ReplayOutputs(); err != nil {
				return
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
		cg.commands[ev.i] = nil
		if cg.deinterlace {
			cg.outputs[ev.i] = nil
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

			// perform hara-kiri
			os.Exit(ev.exitCode)
		}

		if cg.masterId != -1 && cg.masterId == ev.i {
			if cg.deinterlace {
				// dump outputs that are available
				for _, output := range cg.outputs {
					if output != nil {
						_ = output.ReplayOutputs()
					}
				}
			}

			// perform hara-kiri
			os.Exit(ev.exitCode)
		}
	}

	return
}

func (cg *CommandGroup) Add(commandLines ...string) error {
	// some common values for all commands
	env := os.Environ()
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

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
		commands[i] = exec.Command("sh", "-c", commandLines[i])
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
