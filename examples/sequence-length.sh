#!/bin/bash
## sequence-length.sh
##
## @author gdm85
##
## example to show how to run sequential commands in groups
##
#

cat<<EOF | bin/coshell --deinterlace --sequence-length=2
echo alpha
echo beta
echo delta
echo gamma
EOF
