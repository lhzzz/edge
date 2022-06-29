#!/bin/bash
set -e

CMD=cmd/
BIN=bin/
DOCKERFILE=build/docker/
REGISTRY=registry.zhst.com
VERSION=v1.

if [[ $1 == "cicd" ]]
then
    echo "make image in CICD pipeline"
    docker login ${REGISTRY}
    for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
    do 
        if [ "$(ls ${DOCKERFILE}${d})" ]; then 
            cp ${BIN}${d} ${DOCKERFILE}${d}/
            docker build -t ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:${VERSION}${CI_PIPELINE_ID} ${DOCKERFILE}${d}/
            #docker push ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:${VERSION}${CI_PIPELINE_ID}
            rm ${DOCKERFILE}${d}/${d}
            echo ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:${VERSION}${CI_PIPELINE_ID}
        fi
    done
else 
    CI_PROJECT_NAMESPACE=cloud-native
    for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
    do 
        if [ "$(ls ${DOCKERFILE}${d})" ]; then 
            cp ${BIN}${d} ${DOCKERFILE}${d}/
            docker build -t ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:local ${DOCKERFILE}${d}/
            rm ${DOCKERFILE}${d}/${d}
        fi
    done
fi
