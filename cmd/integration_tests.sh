#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

craft version

craft run testserver

craft list -a

craft configure test --prop gamemode=creative

mode=$(docker exec test cat /bedrock/server.properties | grep gamemode)
if [ "$mode" != "gamemode=creative" ]; then
  exit 1
fi

craft stop testserver

craft start testserver

craft backup testserver

craft cmd testserver time set 0600

craft stop testserver