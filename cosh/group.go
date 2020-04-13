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
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type CommandGroup struct {
	commands []*exec.Cmd

	sync.Mutex
	finished []bool
	started  []bool
}

func (cp *CommandPool) NewCommandGroup(cwd string, env []string, stdout, stderr io.Writer, commandLines []string) (*CommandGroup, error) {
	var cg CommandGroup
	l := len(commandLines)
	cg.commands = make([]*exec.Cmd, l)
	cg.finished = make([]bool, l)
	cg.started = make([]bool, l)
	for j, commandLine := range commandLines {
		cmd, err := cp.prepareCommand(commandLine)
		if err != nil {
			// will only happen in case of problems at splitting the command line
			return nil, err
		}
		cg.commands[j] = cmd

		cmd.Env = env
		cmd.Dir = cwd
		cmd.Stdout, cmd.Stderr = stdout, stderr
		// notice here how no stdin is attached to commands
	}

	return &cg, nil
}

func (cg *CommandGroup) setFinished(i int) {
	cg.Lock()
	cg.started[i] = false
	cg.finished[i] = true
	cg.Unlock()
}

// Run will synchronously run all the commands of the command group.
func (cg *CommandGroup) Run() (int, error) {
	if cg == nil {
		panic("BUG: cg is nil")
	}
	for i := range cg.commands {
		cg.Lock()
		err := cg.commands[i].Start()
		if err != nil {
			// always invalidate command after exit
			cg.finished[i] = true
			cg.Unlock()
			return -1, err
		}
		cg.started[i] = true
		cg.Unlock()

		err = cg.commands[i].Wait()
		// always invalidate command after exit
		cg.setFinished(i)
		if err == nil {
			// exit code is 0, pick next command
			continue
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				//NOTE: this is never expected to happen with exit code 0
				return status.ExitStatus(), nil
			}
			return -1, fmt.Errorf("cannot read exit error status: %w", err)
		}

		// any other error
		return -1, err
	}

	// all commands completed successfully - exit code 0
	return 0, nil
}

func (cg *CommandGroup) terminate() {
	cg.Lock()
	defer cg.Unlock()

	for i, cmd := range cg.commands {
		// already finished
		if cg.finished[i] {
			continue
		}

		// not yet started
		if !cg.started[i] {
			continue
		}

		if cmd.Process == nil {
			panic("BUG: unexpected process missing after call to Stat")
		}

		err := cmd.Process.Kill()
		if err != nil && err.Error() != "os: process already finished" {
			fmt.Fprintf(os.Stderr, "ERROR: could not kill process %d: %v\n", cmd.Process.Pid, err)
		}
	}
}
