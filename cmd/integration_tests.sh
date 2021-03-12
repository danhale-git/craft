#!/usr/bin/env bash
go install github.com/danhale-git/craft
docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

# TODO: Remove sleep 2 when issue 40 is resolved: https://github.com/danhale-git/craft/issues/40
sleep 2; echo "craft version"
if ! craft version; then
  exit 1; fi

sleep 2; echo "craft run testserver"
if ! craft run testserver; then
  exit 1; fi

if ! listOut=$(craft list); then
  exit 1; fi
if [[ "$listOut" != "testserver   running - port 19132" ]]; then
  exit 1
fi

sleep 2; echo "craft configure testserver --prop gamemode=creative"
if ! craft configure testserver --prop gamemode=creative; then
  exit 1; fi

if ! mode=$(docker exec testserver cat /bedrock/server.properties | grep gamemode); then
  exit 1; fi
if [[ "$mode" != "gamemode=creative" ]]; then
  exit 1
fi

sleep 2; echo "craft stop testserver"
if ! craft stop testserver; then
  exit 1; fi

sleep 2; echo "craft list -a"
if ! listAllOut=$(craft list -a); then
  exit 1; fi
if [[ $listAllOut != testserver* ]]; then
  exit 1
fi

sleep 2; echo "craft start testserver"
if ! craft start testserver; then
  exit 1; fi

sleep 2; echo "craft backup testserver"
if ! craft backup testserver; then
  exit 1; fi

sleep 2; echo "craft cmd testserver time set 0600"
if ! craft cmd testserver time set 0600; then
  exit 1; fi

sleep 2; echo "craft export testserver -d ~"
if ! craft export testserver -d /; then
  exit 1; fi

sleep 2; echo "craft stop testserver"
if ! craft stop testserver; then
  exit 1; fi
