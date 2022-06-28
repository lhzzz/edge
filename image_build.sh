# !/bin/bash

CMD=cmd/
BIN=bin/
DOCKERFILE=build/docker/
REGISTRY=registry.sakura.com
CI_PROJECT_NAMESPACE=cloud-native
CI_PIPELINE_ID=1

for d in $(ls $CMD -l | grep ^d | awk '{print $9}')
do 
    if [ "$(ls ${DOCKERFILE}${d})" ]; then 
        cp ${BIN}${d} ${DOCKERFILE}${d}/
        docker build -t ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:v1.${CI_PIPELINE_ID} ${DOCKERFILE}${d}/
        docker push ${REGISTRY}/${CI_PROJECT_NAMESPACE}/${d}:v1.${CI_PIPELINE_ID}
        rm ${DOCKERFILE}${d}/${d}
    fi
done
