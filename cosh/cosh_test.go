/*
 * coshell v0.1.5 - a no-frills dependency-free replacement for GNU parallel
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
	"testing"
)

type OptionsCombination struct {
	Deinterlace bool
	Halt bool
}

var (
	combinations = []OptionsCombination{
		{true,true},
		{false,true},
		{true,false},
		{false,false},
	}
	testCommandLines = []string{"echo alpha >/dev/null", "echo beta >/dev/null", "echo gamma >/dev/null", "echo delta >/dev/null"}
)

func TestCommandGroupOptions(t *testing.T) {
	for _, c := range combinations {
		for masterId := -1; masterId<len(testCommandLines); masterId++ {
			var exitCode int
			cg := NewCommandGroup(c.Deinterlace, c.Halt, masterId)
			err := cg.Add(testCommandLines...)
			if err != nil {
				t.Fatal(err.Error())
			}
			err = cg.Start()
			if err != nil {
				t.Fatal(err.Error())
			}
			err, exitCode = cg.Join()
			if err != nil {
				t.Fatal(err.Error())
			}
			if exitCode != 0 {
				t.Fatal("non-zero exit")
			}
		}
	}
}
