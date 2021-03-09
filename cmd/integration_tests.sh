#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

craft run testserver
craft list -a
craft stop testserver
craft start testserver
craft backup testserver
craft cmd testserver time set 0600