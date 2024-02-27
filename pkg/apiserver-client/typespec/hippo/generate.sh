#!/usr/bin/env bash


# please install protoc https://github.com/protocolbuffers/protobuf/releases 
# and golang protobuf http://gitlab.alibaba-inc.com/go-third/protobuf.git

protoc --gogo_out=. --deepcopy_out=. *.proto
protoc --gofast_out=.  --deepcopy_out=. Common.proto
