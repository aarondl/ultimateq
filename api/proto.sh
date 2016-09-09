#!/usr/bin/env sh
protoc -I ./ *.proto --gofast_out=plugins=grpc:./
#protoc -I ./ *.proto --go_out=plugins=grpc:./
