/*
 * coshell v0.1.2 - a no-frills dependency-free replacement for GNU parallel
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
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	signal      syscall.Signal
}

func NewCommandGroup(deinterlace, halt bool, signal syscall.Signal) *CommandGroup {
	return &CommandGroup{deinterlace: deinterlace, halt: halt, signal: signal}
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
	for {
		ev := <-chainOfEvents
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

		// mark command as finished
		cg.commands[ev.i] = nil

		// broadcast a signal to all neighbours
		if cg.halt && ev.exitCode != 0 {
			for i := 0; i < len(cg.commands); i++ {
				if i == ev.i || cg.commands[i] == nil {
					// skip self and already completed processes
					continue
				}

				// send signal, even if process is no more running
				go func(i int) {
					err := cg.commands[i].Process.Signal(cg.signal)
					if err != nil {
						// print error and keep sending signals
						fmt.Fprintf(os.Stderr, "kill: %v", err)
					}
				}(i)
			}

			// do not repeat signal-sending
			cg.halt = false
		}
	}

	return
}

func parseTokens(s string) (tokens []string, err error) {
	s = strings.TrimSpace(s)

	escaping := false
	quoting := false
	squoting := false
	just_quoted := false
	accum := ""
	for i := 0; i < len(s); i++ {
		if escaping {
			// escape sequences
			if s[i] == '\\' {
				accum += fmt.Sprintf("%c", s[i])
			} else {
				if (quoting && s[i] == '"') || (squoting && s[i] == '\'') {
					accum += fmt.Sprintf("%c", s[i])
				} else {
					err = fmt.Errorf("unrecognized escape sequence: '\\%c'", s[i])
					return
				}
			}
			escaping = false

			continue
		}

		if quoting {
			// interrupt double quoting
			if s[i] == '"' {
				quoting = false
				just_quoted = true
				tokens = append(tokens, accum)
				accum = ""
				continue
			}
		} else {
			just_quoted = false
		}

		if squoting {
			// interrupt single quoting
			if s[i] == '\'' {
				squoting = false
				just_quoted = true
				tokens = append(tokens, accum)
				accum = ""
				continue
			}
		} else {
			just_quoted = false
		}

		switch s[i] {
		case '\\':
			if quoting || squoting {
				escaping = true
			}
		case '"':
			if squoting {
				accum += fmt.Sprintf("%c", s[i])
			} else {
				quoting = true
			}
		case '\'':
			if quoting {
				accum += fmt.Sprintf("%c", s[i])
			} else {
				squoting = true
			}
		case ' ':
			if quoting || squoting {
				accum += fmt.Sprintf("%c", s[i])
			} else {
				if len(accum) == 0 {
					if just_quoted {
						tokens = append(tokens, accum)
					}
				} else {
					tokens = append(tokens, accum)
					accum = ""
				}
			}
		default:
			accum += fmt.Sprintf("%c", s[i])
		}
	}

	if quoting {
		err = errors.New("unterminated double-quoted string")
		return
	}

	if squoting {
		err = errors.New("unterminated single-quoted string")
		return
	}

	if escaping {
		err = errors.New("unterminated escape sequence")
		return
	}

	if len(accum) > 0 {
		tokens = append(tokens, accum)
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
		tokens, err := parseTokens(commandLines[i])
		if err != nil {
			return err
		}
		if len(tokens) == 0 || len(tokens[0]) == 0 {
			return fmt.Errorf("line %d: invalid command line", i)
		}
		commands[i] = exec.Command(tokens[0], tokens[1:]...)
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
	cg.outputs = append(cg.outputs, outputs...)

	return nil
}
