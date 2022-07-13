#!/bin/bash

kubectl apply -f edge-namespace.yaml

helmver=$(helm version | grep v3)
echo $helmver
if [[ -z $helmver ]]
then
#helm 2
    helm install edge_registry/  --name=edge-registry --namespace=edge-cluster
else
#helm 3
    helm install edge_registry/ --name-template=edge-registry --namespace=edge-cluster
fi
