#!/usr/bin/env bash
go install github.com/danhale-git/craft

export PATH="$PATH:~/go/bin/"

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

echo "craft configure testserver --prop allow-cheats=true"
if ! craft configure testserver --prop allow-cheats=true; then
  exit 1; fi

echo "craft run testserver2"
if ! craft run testserver2; then
  exit 1; fi

echo "craft stop testserver testserver2"
if ! craft stop testserver testserver2; then
  exit 1; fi

echo "craft list -a"
if ! listAllOut=$(craft list -a); then
  exit 1; fi
if [[ $listAllOut != testserver* ]]; then
  exit 1
fi

echo "craft start testserver testserver2"
if ! craft start testserver testserver2; then
  exit 1; fi

echo "craft backup testserver testserver2"
if ! craft backup testserver testserver2; then
  exit 1; fi

echo "craft cmd testserver time set 0600"
if ! craft cmd testserver time set 0600; then
  exit 1; fi

echo "craft export testserver2 -d ~"
if ! craft export testserver2; then
  exit 1; fi

echo "craft stop testserver testserver2"
if ! craft stop testserver testserver2; then
  exit 1; fi
