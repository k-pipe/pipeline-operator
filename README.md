# pipeline-operator

Follow tutorial: https://sdk.operatorframework.io/docs/building-operators/helm/quickstart/

```
operator-sdk init --domain kpipe --plugins helm 
operator-sdk create api --group demo --version v1alpha1 --kind Nginx
make docker-build docker-push IMG="kpipe/nginx-operator:v0.0.1"
operator-sdk olm install
make bundle IMG="kpipe/nginx-operator:v0.0.1"  # give some input here
make bundle-build bundle-push IMG="kpipe/nginx-operator:v0.0.1"
```

To use it in kpipe test cluster

```
gcloud config set account j@kneissler.com
gcloud auth application-default set-quota-project k-pipe-test-system
gcloud config set project k-pipe-test-system
gcloud container clusters get-credentials k-pipe-runner --region europe-west3
gcloud projects add-iam-policy-binding k-pipe-test-system \
  --member=user:j@kneissler.com \
  --role=roles/container.admin
make deploy IMG="kpipe/nginx-operator:v0.0.1"
```



## Creation 

This repository was setup using the go kubernetes operator framework by running the following commands (on a mac obviously):

```
brew install go operator-sdk
```

On april 17 2024 this installed go 1.22.2 and operator-sdk 1.34.1

Next:

```
go mod init example.com/m/v2
```

This created a simple `go.mod`.

```
operator-sdk init —domain k-pipe.cloud —repo github.com/k-pipe/pipeline-operator.git —plugins=go/v4-alpha
```

This extended [go.mod](./go.mod) (setting go version down to 1.20 and adding a bunch of dependencies).
Furthermore it created the following files and sub-folders:
 * [.dockerignore](./.dockerignore)
 * [.gitignore](./.gitignore)
 * [.golangci.yml](./.golangci.yml)
 * [Dockerfile](./Dockerfile)
 * [go.sum](./go.sum)
 * [Makefile](./Makefile)
 * [PROJECT](./PROJECT)
 * [cmd](./cmd)
 * [config](./config)
 * [hack](./hack)
 * [test](./test)

The following command was used to create a CRD for the pipeline resource:

```
operator-sdk create api --group apps --version v1alpha1 --kind Pipeline --resource --controller
```

## Background

As introduction to kubernetes operators can be found here: https://shahin-mahmud.medium.com/write-your-first-kubernetes-operator-in-go-177047337eae
Another medium article that was helpful in setting this up: https://www.faizanbashir.me/guide-to-create-kubernetes-operator-with-golang

Note: example operator hosted as helm chart on github: https://backaged.github.io/tdset-operator/