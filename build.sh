#! /bin/bash

APIDIR=$PWD/api/edge-proto
chmod +x $APIDIR/*
cd api/edge-proto
./make_pb.sh $APIDIR/pb $APIDIR/proto
cd -

go build -o bin/edgelet cmd/edgelet/main.go &
go build -o bin/edgectl cmd/edgectl/main.go &
go build -o bin/edge-registry cmd/edge-registry/main.go &
wait

ls -l bin/