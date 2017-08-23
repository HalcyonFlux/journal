#!/bin/sh

if [ `which protoc` -eq '' ]; then
	echo "Protoc (protobuf compiler) not found!"
fi

protoc --go_out=plugins=grpc:. ../protobuf/log.proto

if [ $? -ne 0 ]; then
	echo "Compilation failed. Aborting";
	exit 1;
fi

go install github.com/vaitekunas/log/cmd/log

if [ $? -ne 0 ]; then
	echo "Installation failed. Aborting"
	exit 1;
fi

echo "Log binary installed"
