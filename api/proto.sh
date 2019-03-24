#!/usr/bin/env sh
protoc -I ./ *.proto --go_out=plugins=grpc:./
