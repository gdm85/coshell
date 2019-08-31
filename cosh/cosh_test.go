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
	"fmt"
	"testing"
)

var (
	testCommandLinesWithShell    []string
	testCommandLinesWithoutShell []string
)

func init() {
	for i := 0; i < 26; i++ {
		testCommandLinesWithShell = append(testCommandLinesWithShell, "echo -n")
		testCommandLinesWithShell = append(testCommandLinesWithShell, "cd /tmp")

		testCommandLinesWithoutShell = append(testCommandLinesWithoutShell, "sh -c \"echo -n\"")
		testCommandLinesWithoutShell = append(testCommandLinesWithoutShell, fmt.Sprintf("sh -c \"cd /tmp\""))
	}
}

func TestCommandGroupOptions(t *testing.T) {
	for _, shellArgs := range [][]string{nil, []string{"sh", "-c"}} {
		shellArgs := shellArgs
		testCommandLines := testCommandLinesWithShell
		if len(shellArgs) == 0 {
			testCommandLines = testCommandLinesWithoutShell
		}
		for _, deinterlace := range []bool{true, false} {
			deinterlace := deinterlace
			for _, halt := range []bool{true, false} {
				halt := halt
				for _, ordered := range []bool{true, false} {
					ordered := ordered
					for _, jobs := range []int{0, 16, 32} {
						jobs := jobs
						for masterId := -1; masterId < len(testCommandLines)/2; masterId++ {
							masterId := masterId

							name := fmt.Sprintf("s=%v d=%v h=%v o=%v j=%d m=%d", shellArgs, deinterlace, halt, ordered, jobs, masterId)

							t.Run(name, func(t *testing.T) {
								t.Parallel()
								var exitCode int
								cg := NewCommandGroup(shellArgs, deinterlace, halt, masterId, ordered)
								err := cg.Add(testCommandLines...)
								if err != nil {
									t.Fatal(err.Error())
								}
								err = cg.Start(jobs)
								if err != nil {
									t.Fatal(err.Error())
								}
								exitCode, err = cg.Join()
								if err != nil {
									t.Fatal(err.Error())
								}
								if exitCode != 0 {
									t.Fatal("non-zero exit")
								}
							})
						}
					}
				}
			}
		}
	}
}
