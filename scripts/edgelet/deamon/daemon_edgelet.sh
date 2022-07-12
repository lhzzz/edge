#!/bin/bash

USER=$(whoami)
SUDO="sudo"
if [[ $USER == "root" ]]; then 
    SUDO=""
fi

$SUDO systemctl stop edgelet
$SUDO cp edgelet /usr/bin/
$SUDO cp edgelet.service /etc/systemd/system/
$SUDO systemctl daemon-reload && systemctl restart edgelet.service && systemctl enable edgelet.service