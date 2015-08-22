#!/bin/sh
## deinterlace.sh
##
## @author gdm85
##
## example to show correctly deinterlaced outputs
## lines should be split in blocks of 9 per each command
##
#

generate_testcase() {
	local N
	local AMT
	for N in `seq 10`; do
		for L in `seq 9`; do
			AMT=$(( $RANDOM % 1000 ))
			AMT=$((AMT / 2 ))
			if [ $(( $L % 2 )) -eq 0 ]; then
				echo "sleep 0.$AMT && echo 'Line $L of command $N'"
			else
				echo "sleep 0.$AMT && echo 'Line $L of command $N (stderr)' 1>&2"
			fi
		done
	done
}

generate_testcase | ./coshell -d
