#!/bin/bash

#set -x

. ./build-common

make_go_binary etcdconfs.go
#make_go_binary neardconfs.go
make_go_binary restconfs.go
