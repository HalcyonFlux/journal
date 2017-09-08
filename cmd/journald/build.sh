#!/bin/sh

git describe  | awk '{print "package main\nconst VERSION = \"" $1 "\""}' > version.go

if [ "$#" -eq 0 ] || [ "$1" -eq "build" ]; then
  go build github.com/vaitekunas/journal/cmd/journald
  exit 0
fi

if [ "$1" -eq "install" ]; then
  go install github.com/vaitekunas/journal/cmd/journald
  exit 0
fi
