#!/bin/bash
## pack-leader.sh
##
## @author gdm85
##
## example to show how to use --master option
##
#

generate_testcase() {
	local N
	local AMT
	for N in `seq 0 19`; do
		if [ $N -eq 3 ]; then
			echo "sleep $N && false"
		else
			echo "sleep $N && echo 'slept $N seconds'"
		fi
	done
}

generate_testcase | bin/coshell --deinterlace --master 3
