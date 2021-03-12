#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

# TODO: Remove sleep 2 when issue 40 is resolved: https://github.com/danhale-git/craft/issues/40
sleep 2; echo "craft version"
craft version

sleep 2; echo "craft run testserver"
craft run testserver

listOut=$(craft list)
if [[ "$listOut" != "testserver   running - port 19132" ]]; then
  exit 1
fi

sleep 2; echo "craft configure testserver --prop gamemode=creative"
craft configure testserver --prop gamemode=creative

mode=$(docker exec testserver cat /bedrock/server.properties | grep gamemode)
if [[ "$mode" != "gamemode=creative" ]]; then
  exit 1
fi

sleep 2; echo "craft stop testserver"
craft stop testserver

sleep 2; echo "craft list -a"
listAllOut=$(craft list -a)
if [[ $listAllOut != testserver* ]]; then
  exit 1
fi

sleep 2; echo "craft start testserver"
craft start testserver

sleep 2; echo "craft backup testserver"
craft backup testserver

sleep 2; echo "craft cmd testserver time set 0600"
craft cmd testserver time set 0600

sleep 2; echo "craft export testserver -d ~"
craft export testserver -d /

sleep 2; echo "craft stop testserver"
craft stop testserver