# pipeline-operator


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
operator-sdk init —domain k-pipe.cloud —repo github.com/k-pipe/pipeline-operator.git 
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

## Background

As introduction to kubernetes operators can be found here: https://shahin-mahmud.medium.com/write-your-first-kubernetes-operator-in-go-177047337eae
Another medium article that was helpful in setting this up: https://www.faizanbashir.me/guide-to-create-kubernetes-operator-with-golang

