# k-pipe Operator

This provides a kubernetes operator to define, run and schedule pipelines. 

## Getting started using helm

On a kubernetes cluster of your choice install the operator using the following commands:

```
helm repo add k-pipe https://k-pipe.github.io/pipeline-operator/
helm install k-pipe k-pipe/tdset-controller -n tdset
```

You might then do ....


## Further information

The operator is build in this [github project](https://github.com/k-pipe/pipeline-operator).
Further technical details can be found there.
