#!/usr/bin/env sh
protoc -I ./ ultimateq.proto --go_out=plugins=grpc:./
