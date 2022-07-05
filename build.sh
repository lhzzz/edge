#! /bin/bash
set -e

APIDIR=$PWD/api/edge-proto
chmod +x $APIDIR/*
cd api/edge-proto
./make_pb.sh $APIDIR/pb $APIDIR/proto
cd -

CMD=cmd/
GOARCH=$(go env GOARCH)
echo "building in "$GOARCH

for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
do
{
    go build -o bin/${GOARCH}/${d} -ldflags="-X main.buildVersion=v${CI_PIPELINE_ID}" cmd/${d}/main.go
}&
done
wait

ls -l bin/$GOARCH/