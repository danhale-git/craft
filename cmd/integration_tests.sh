#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

~/go/bin/craft run testserver1