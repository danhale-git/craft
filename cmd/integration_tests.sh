#!/usr/bin/env bash
go install github.com/danhale-git/craft
#docker pull danhaledocker/craftmine:v1.7

export PATH=$PATH:~/go/bin/

echo "craft build"
if ! craft build; then
  exit 1; fi

echo "craft version"
if ! craft version; then
  exit 1; fi

echo "craft run testserver"
if ! craft run testserver; then
  exit 1; fi

if ! listOut=$(craft list); then
  exit 1; fi
if [[ "$listOut" != "testserver   running - port 19132" ]]; then
  exit 1
fi

echo "craft configure testserver --prop gamemode=creative"
if ! craft configure testserver --prop gamemode=creative; then
  exit 1; fi

if ! mode=$(docker exec testserver cat /bedrock/server.properties | grep gamemode); then
  exit 1; fi
if [[ "$mode" != "gamemode=creative" ]]; then
  exit 1
fi

echo "craft stop testserver"
if ! craft stop testserver; then
  exit 1; fi

echo "craft list -a"
if ! listAllOut=$(craft list -a); then
  exit 1; fi
if [[ $listAllOut != testserver* ]]; then
  exit 1
fi

echo "craft start testserver"
if ! craft start testserver; then
  exit 1; fi

echo "craft backup testserver"
if ! craft backup testserver; then
  exit 1; fi

echo "craft cmd testserver time set 0600"
if ! craft cmd testserver time set 0600; then
  exit 1; fi

echo "craft export testserver -d ~"
if ! craft export testserver; then
  exit 1; fi

echo "craft stop testserver"
if ! craft stop testserver; then
  exit 1; fi
