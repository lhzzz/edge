# !/bin/bash

outputPath=$1
protoPath=$2

protoc -I$protoPath --go_out=plugins=grpc:$outputPath $protoPath/*.proto