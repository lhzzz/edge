# !/bin/bash

outputPath=$1
protoPath=$2

cmd="protoc "
for d in $(ls -l $protoPath | grep ^d | grep -v tests | awk '{print $9}')
do    
    cmd=$cmd" -I"$protoPath"/"$d
done
echo "proto dest dir : " $1
cmd=$cmd" --go_out=plugins=grpc:"$1" "
for f in $(find $protoPath -name "*.proto" -exec basename {} \;)
do
    cmd=$cmd" "$f
done
echo $cmd
$cmd

