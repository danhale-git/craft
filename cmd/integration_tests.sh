#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

craft run testserver1
craft list -a