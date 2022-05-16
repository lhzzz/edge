# !/bin/bash

cp bin/edge-registry build/docker/edge-registry/
cd build/docker/edge-registry/
docker build -t registry.sakura.com/edge-registry:v1.0 .
rm edge-registry
cd -
