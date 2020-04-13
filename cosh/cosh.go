/*
 * coshell v0.2.3 - a no-frills dependency-free replacement for GNU parallel
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

package cosh

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

var ErrEmptyCommandLine = errors.New("empty command line")

type event struct {
	index    int
	err      error
	exitCode int
}

type CommandPoolConfig struct {
	Deinterlace bool
	Halt        bool
	MasterID    int
	ShellArgs   []string
	Stdout      io.Writer
	Stderr      io.Writer
}

type CommandPool struct {
	groups          []*CommandGroup
	outputs         []*SortedOutput
	completedGroups chan event

	CommandPoolConfig
}

var DefaultCommandPoolConfig = CommandPoolConfig{
	Stderr:   os.Stderr,
	Stdout:   os.Stdout,
	MasterID: -1,
}

func NewCommandPool(cfg *CommandPoolConfig) *CommandPool {
	if cfg == nil {
		cfg = &DefaultCommandPoolConfig
	}
	return &CommandPool{
		CommandPoolConfig: *cfg,
	}
}

// Add will add the specified command lines grouped by sequence length.
// Each group will run sequentially and require that the previous command is successful.
func (cp *CommandPool) Add(sequenceLength int, commandLines ...string) error {
	// some common values for all commands
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	env := os.Environ()

	// prepare command groups to be executed sequentially
	l := len(commandLines) / sequenceLength
	cp.groups = make([]*CommandGroup, l)
	if cp.Deinterlace {
		cp.outputs = make([]*SortedOutput, l)
	}
	for i := 0; i < l; i++ {
		var stdout, stderr io.Writer
		if cp.Deinterlace {
			cp.outputs[i] = NewSortedOutput(cp.Stdout, cp.Stderr)
			stdout, stderr = cp.outputs[i].stdout, cp.outputs[i].stderr
		} else {
			stdout, stderr = cp.Stdout, cp.Stderr
		}
		cp.groups[i], err = cp.NewCommandGroup(cwd, env, stdout, stderr, commandLines[i*sequenceLength:(i+1)*sequenceLength])
		if err != nil {
			return err
		}
	}

	return nil
}

// Start will start all command groups concurrently.
func (cp *CommandPool) Start(jobs int) error {
	if jobs == 0 {
		// set to maximum possible
		jobs = len(cp.groups)
	}
	res := make(chan struct{}, jobs)
	for i := 0; i < jobs; i++ {
		res <- struct{}{}
	}

	cp.completedGroups = make(chan event, len(cp.groups))

	// run all groups concurrently
	for i := 0; i < len(cp.groups); i++ {
		go func(i int) {
			<-res

			exitCode, err := cp.groups[i].Run()
			cp.completedGroups <- event{
				index:    i,
				err:      err,
				exitCode: exitCode,
			}

			res <- struct{}{}
		}(i)
	}

	return nil
}

// Join waits for all command groups to complete execution and return the (unsigned) sum of each individual exit code.
func (cp *CommandPool) Join() (int, error) {
	l := len(cp.groups)

	displayedSoFar := 0
	var outputErrors []error
	var exitCode uint
	var exitSelected bool

	for count := 0; count < l; count++ {
		ev := <-cp.completedGroups

		if cp.Deinterlace {
			// print deinterlaced output on the go
			if displayedSoFar == ev.index {
				if err := cp.outputs[ev.index].ReplayOutputs(); err != nil {
					outputErrors = append(outputErrors, err)
				}
				displayedSoFar++

				// will not be accessed anymore
				cp.outputs[ev.index] = nil
			}
		}

		// an unexpected error during wait and exit code processing
		if ev.err != nil {
			cp.terminateAll(ev.index)
			//NOTE: not waiting for processes to terminate
			return -1, ev.err
		}

		if exitSelected {
			continue
		}

		// some process terminated, spool outputs and use its exit code
		if (cp.Halt && ev.exitCode != 0) ||
			// master process exited, terminate all and use its exit code
			(cp.MasterID != -1 && cp.MasterID == ev.index) {

			cp.terminateAll(ev.index)

			exitCode = uint(ev.exitCode)
			exitSelected = true
			continue
		}

		// regular processes
		exitCode += uint(ev.exitCode)
	}

	// print remaining unsorted outputs
	if cp.Deinterlace {
		for i := displayedSoFar; i < l; i++ {
			if err := cp.outputs[i].ReplayOutputs(); err != nil {
				outputErrors = append(outputErrors, err)
			}
		}
		if len(outputErrors) != 0 {
			fmt.Fprintf(os.Stderr, "ERROR: %d errors while replaying output:\n%v\n", len(outputErrors), outputErrors)
		}
	}

	// convert back exit code from unsigned integer
	return int(exitCode), nil
}

func (cp *CommandPool) prepareCommand(cmdLine string) (*exec.Cmd, error) {
	// using a shell prefix, append the whole command line
	if len(cp.ShellArgs) != 0 {
		return exec.Command(cp.ShellArgs[0], append(cp.ShellArgs[1:], cmdLine)...), nil
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

func (cp *CommandPool) terminateAll(exceptIndex int) {
	var wg sync.WaitGroup
	for i, cg := range cp.groups {
		if i == exceptIndex {
			continue
		}

		wg.Add(1)
		go func(cg *CommandGroup) {
			cg.terminate()
			wg.Done()
		}(cg)
	}

	wg.Wait()
}
