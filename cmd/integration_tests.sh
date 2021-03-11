#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

# TODO: Remove sleep 2 when issue 40 is resolved: https://github.com/danhale-git/craft/issues/40
sleep 2; craft version

sleep 2; craft run testserver

listOut=$(craft list)
if [[ "$listOut" != "testserver   running - port 19132" ]]; then
  exit 1
fi

sleep 2; craft configure testserver --prop gamemode=creative

mode=$(docker exec testserver cat /bedrock/server.properties | grep gamemode)
if [[ "$mode" != "gamemode=creative" ]]; then
  exit 1
fi

sleep 2; craft stop testserver

listAllOut=$(sleep 2; craft list -a)
if [[ $listAllOut != testserver* ]]; then
  exit 1
fi

sleep 2; craft start testserver

sleep 2; craft backup testserver

sleep 2; craft cmd testserver time set 0600

sleep 2; craft export testserver -d /

sleep 2; craft stop testserver