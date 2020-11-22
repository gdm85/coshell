/*
 * coshell v0.2.5 - a no-frills dependency-free replacement for GNU parallel
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
	"bytes"
	"io"
	"sync"
)

const (
	// OutputStdout corresponds to os.Stdout.
	OutputStdout = iota
	// OutputStderr corresponds to os.Stderr.
	OutputStderr
)

// OutputType is the type of output.
type OutputType int

// SortedOutput contains state to replay output segments in the same order
// they were received in.
type SortedOutput struct {
	sync.Mutex
	stdout SortedOutputWriter
	stderr SortedOutputWriter

	buffer bytes.Buffer

	segments     []segment
	parentStdout io.Writer
	parentStderr io.Writer
}

// SortedOutputWriter is a writer for sorted output.
type SortedOutputWriter struct {
	outputType OutputType
	parent     *SortedOutput
}

type segment struct {
	outputType OutputType
	offset     int
	length     int
}

// NewSortedOutput constructs a new SortedOutput.
func NewSortedOutput(stdout, stderr io.Writer) *SortedOutput {
	sp := SortedOutput{
		parentStdout: stdout,
		parentStderr: stderr,
	}
	sp.stdout = SortedOutputWriter{OutputStdout, &sp}
	sp.stderr = SortedOutputWriter{OutputStderr, &sp}

	return &sp
}

// ReplayOutputs will replay all stdout/stderr outputs to parent stdout/stderr.
func (so *SortedOutput) ReplayOutputs() error {
	so.Lock()
	defer so.Unlock()
	data := so.buffer.Bytes()

	for _, segment := range so.segments {
		if segment.outputType == OutputStdout {
			if _, err := so.parentStdout.Write(data[segment.offset : segment.offset+segment.length]); err != nil {
				return err
			}
			continue
		}
		// if it's not stdout, then it's stderr
		if _, err := so.parentStderr.Write(data[segment.offset : segment.offset+segment.length]); err != nil {
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
