#!/bin/bash

USER=$(whoami)
DOCKER_CONFIG="/home/"$USER"/.docker"
EDGELET_IMAGE=registry.edge.com/cloud-native/edgelet:local
SUDO="sudo"
if [[ $USER == "root" ]]; then 
    SUDO=""
fi

if [[ $($SUDO ls /data/edgelet) ]]
then
    echo "alreay has dir /data/edgelet"
else
    $SUDO mkdir -p /data/edgelet/
fi 

docker run -it \
    -v /data/edgelet:/data/edgelet/ \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $DOCKER_CONFIG:/root/.docker \
    --network host \
    --restart=always \
    $EDGELET_IMAGE /bin/bash
