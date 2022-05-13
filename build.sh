#! /bin/bash

apiDir=$PWD/api/
cd api/
./make_pb.sh $apiDir/pb $apiDir/proto
cd ..

go build -o bin/edgelet cmd/edgelet/main.go &
go build -o bin/edgectl cmd/edgectl/main.go &
go build -o bin/edge-registry cmd/edge-registry/main.go &
wait

ls -l bin/