#!/bin/bash 

# curl -L "https://github.com/docker/compose/releases/download/v2.5.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

cp docker-compose-$(uname -s)-$(uname -m) /usr/local/bin/docker-compose

