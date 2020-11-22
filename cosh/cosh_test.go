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
	"fmt"
	"testing"
)

var (
	testCommandLinesWithShell    []string
	testCommandLinesWithoutShell []string
)

const maxCommands = 16

func init() {
	for i := 0; i < maxCommands; i++ {
		testCommandLinesWithShell = append(testCommandLinesWithShell, "echo -n")
		testCommandLinesWithShell = append(testCommandLinesWithShell, "cd /tmp")

		testCommandLinesWithoutShell = append(testCommandLinesWithoutShell, "sh -c \"echo -n\"")
		testCommandLinesWithoutShell = append(testCommandLinesWithoutShell, "sh -c \"cd /tmp\"")
	}
}

func TestCommandPoolOptions(t *testing.T) {
	t.Parallel()

	for _, shellArgs := range [][]string{nil, []string{"sh", "-c"}} {
		shellArgs := shellArgs
		testCommandLines := testCommandLinesWithShell
		hasShellArgs := 1
		if len(shellArgs) == 0 {
			testCommandLines = testCommandLinesWithoutShell
			hasShellArgs = 0
		}
		for _, deinterlace := range []bool{true, false} {
			deinterlace := deinterlace
			for _, halt := range []bool{true, false} {
				halt := halt
				for _, jobs := range []int{0, maxCommands / 6, maxCommands / 2} {
					jobs := jobs

					for _, seqLen := range []int{1, 2, 4} {
						seqLen := seqLen

						for masterID := -1; masterID < len(testCommandLines)/2; masterID++ {
							masterID := masterID

							name := fmt.Sprintf("shellArgs=%v d=%v h=%v j=%d m=%d seqLen=%d", hasShellArgs, deinterlace, halt, jobs, masterID, seqLen)

							t.Run(name, func(t *testing.T) {
								t.Parallel()

								cfg := DefaultCommandPoolConfig
								cfg.ShellArgs = shellArgs
								cfg.Deinterlace = deinterlace
								cfg.Halt = halt
								cfg.MasterID = masterID

								var exitCode int
								cg := NewCommandPool(&cfg)
								err := cg.Add(seqLen, testCommandLines...)
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
