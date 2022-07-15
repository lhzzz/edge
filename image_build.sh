#!/bin/bash
set -e

CMD=cmd/
BIN=bin
DOCKERFILE=build/docker
REGISTRY=registry.zhst.com
VERSION=v

if [[ $1 == "cicd" ]]
then
    echo "make image in CICD pipeline"
    docker login ${REGISTRY}
    for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
    do 
    {   if [ "$(ls ${DOCKERFILE}/${d})" ]; then 
            PLATFORM=""
            if [ "$(ls ${BIN}/amd64/${d})" ]; then 
                PLATFORM="linux/amd64"
                cp ${BIN}/amd64/${d} ${DOCKERFILE}/${d}/mutiple/amd64
            fi
            if  [ "$(ls ${BIN}/arm64/${d})" ]; then 
                PLATFORM=$PLATFORM",linux/arm64"
                cp ${BIN}/arm64/${d} ${DOCKERFILE}/${d}/mutiple/arm64
            fi
            LATEST=${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:latest
            docker buildx build --platform ${PLATFORM} -t ${LATEST} -t ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:${VERSION}${CI_PIPELINE_ID} ${DOCKERFILE}/${d}/mutiple/ --push
        fi
    }
    done
    for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
    do
        echo ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:${VERSION}${CI_PIPELINE_ID}
    done
else 
    CI_PROJECT_NAMESPACE=cloud-native
    GOARCH=$(go env GOARCH)
    for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
    do 
        if [ "$(ls ${DOCKERFILE}/${d})" ]; then 
            cp ${BIN}/${GOARCH}/${d} ${DOCKERFILE}/${d}/single
            docker build -t ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:local ${DOCKERFILE}/${d}/single/
            rm ${DOCKERFILE}/${d}/single/${d}
        fi
    done
fi
