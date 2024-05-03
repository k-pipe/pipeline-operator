# k-pipe Operator

This provides a kubernetes operator to define, run and schedule pipelines. 

## Getting started with helm

```
helm repo add k-pipe https://k-pipe.github.io/pipeline-operator/
helm install k-pipe k-pipe/tdset-controller -n tdset
```


## Background

The operator is based on the operator framework (Go version): https://github.com/operator-framework/operator-sdk
The setup is inspired by this excellent introduction to kubernetes operators: https://shahin-mahmud.medium.com/write-your-first-kubernetes-operator-in-go-177047337eae
Another medium article that was helpful: https://www.faizanbashir.me/guide-to-create-kubernetes-operator-with-golang

Note: example operator hosted as helm chart on github: https://backaged.github.io/tdset-operator/
