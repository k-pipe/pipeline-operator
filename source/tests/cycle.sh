#!/bin/sh
echo uninstall chart
kc delete crd pipelineschedules.pipeline.k-pipe.cloud
helm uninstall k-pipe -n k-pipe
echo remove repo
helm repo remove k-pipe
echo add repo
helm repo add k-pipe https://k-pipe.github.io/pipeline-operator/
echo install chart
helm install k-pipe k-pipe/pipeline-controller -n k-pipe
kubectl delete deployment k-pipe-pipeline-controller -n k-pipe