#!/bin/bash
## default.sh
##
## @author gdm85
##
## example to show default options behaviour
##
#

generate_testcase() {
	local N
	local AMT
	for N in `seq 1 7`; do
		echo "sleep $N && echo 'slept $N seconds'"
	done
}

generate_testcase | bin/coshell
