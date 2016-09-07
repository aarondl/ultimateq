#!/usr/bin/env sh
protoc -I ./ *.proto --gofast_out=plugins=grpc:./
