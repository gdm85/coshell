#!/bin/bash
## deinterlace.sh
##
## @author gdm85
##
## example to show how to terminate neighbour processes when first fails
##
#

generate_testcase() {
	local N
	local AMT
	for N in `seq 20`; do
		if [ $N -eq 4 ]; then
			echo "sh -c 'sleep 3 && false'"
		else
			echo "sleep $N"
		fi
	done
}

generate_testcase | ./coshell --deinterlace --halt
