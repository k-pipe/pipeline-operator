# pipeline-operator

This repository builds the necessary ingredients for a kubernetes operator for processing pipelines:

 * the docker image running the operator code ([result here](https://hub.docker.com/repository/docker/kpipe/pipeline-operator/general))
 * a helm chart with a deployment of the operator, crd definitions and necessary RBAC definitions ([result here](https://k-pipe.github.io/pipeline-operator/))

Documentation and usage examples of the pipeline operator can be found in the 
[README of branch helm](https://github.com/k-pipe/pipeline-operator/blob/helm/README.md).

## Build-in-branches strategy

Different stages of the developemnt/build cycles are done in separate branches. This allows separation of 
primary sources, specification of helm chart and local development (using the secondary sources generated by 
the operator-sdk framework).

### Branch main 

All primary sources required to build and test the operator can be found in the `main` branch. Those are:
 
 * [build script](build-and-push.sh): script which is run on pushes to branch main (see [build.yml](.github/workflows/build.yml))
 * [src/api](https://github.com/k-pipe/pipeline-operator/tree/main/source/api): go files that define the data types for the various CRDs
 * [src/controller](https://github.com/k-pipe/pipeline-operator/tree/main/source/controller): go files that implement the reconcilation logics
 * [src/tests](https://github.com/k-pipe/pipeline-operator/tree/main/source/tests): scripts useful for testing the operator on a kubernetes cluster
 * [version](version): an automatically updated text file that holds the current release version

During building on the main branch, generated files will be committed and pushed to the branches `helm` and `generated`.
The build process of the main branch consists of the following stages:
 
 * fetch other branches and prepare git for pushing
 * tag with version from file `version`
 * install go and operator sdk
 * init the operator, create required APIs
 * overwrite generated go template files with their counterparts from `src/api`
 * generate crd manifests
 * commit and push crd definitions to `helm` branch (after required version substitutions)
 * overwrite generated go template files with their counterparts from `src/controller`
 * building and pushing docker image
 * commit and push all generated files (secondary sources) to branch `generated`; the folders `src/api`, `src/tests` are also added to 
   `generated` (under the folder `main`) in order to be able to change them locally in that branch

### Branch helm

The branch `helm` contains the required additional definitions required to release a helm chart. The generated crds 
will be updated by each run of the cicd process of branch `main` (see above).

The folldowing files are expected in branch `helm`:

 * [helm-chart.yml](https://github.com/k-pipe/pipeline-operator/blob/helm/.github/workflows/helm-chart.yml): 
   releases the helm chart using the predefined `chart-releaser-action` 
 * [charts/pipeline/crds](https://github.com/k-pipe/pipeline-operator/tree/helm/charts/tdset/crds): folder with the generated crd definitions
 * [charts/pipeline/templates](https://github.com/k-pipe/pipeline-operator/tree/helm/charts/tdset/templates): folder with yaml files that defined other parts of the helm chart (deployment, service account and roles)
 * [README.md](https://github.com/k-pipe/pipeline-operator/blob/helm/README.md): the documentation of the pipeline operator
 * [version](https://github.com/k-pipe/pipeline-operator/blob/helm/version): the current version (copied from `main` branch)

### Branch gh-pages

The branch `gh-pages` contains the files for a web page that serves a helm chart release. It is automatically updated
in each run of the cicd process of branch `helm`.

The folldowing files are expected in branch `gh-pages`:
 * [index.yml](https://github.com/k-pipe/pipeline-operator/blob/gh-pages/index.yaml): file that lists all available releases of the site
 * [README.md](https://github.com/k-pipe/pipeline-operator/blob/gh-pages/README.md): documentation of the helm package 
   (instructions how to apply the helm chart and use the operator)

### Branch generated

The branch `generate` contains the files generated by the operator-sdk during the build process in branch `main`.
This can be used for local debugging/development (see below). For conveniance, the folders `main/test` and `main/api`
in this branch are mirrors from `main` (and changes to them will be also pushed back to the `main` branch). The source files under `source/controller`
of branch `main`, however, are not mirrored to `main/controller` because they are already contained in `ìnternal/controller`
(from where also changes will be pushed back to branch 'main' automatically).

The [cicd script](https://github.com/k-pipe/pipeline-operator/blob/generated/.github/workflows/push-to-main.yml) of branch `generated`
performs the following actions:

 * compares the two version files in branch `main` and branch `generated`, if they are different the value from branch `generated`
   will be used (because apparently it has been pushed with some changes by a user). If they are equal, the patch number
   will be incremented by 1 (because every a helm release requires a unique version).
 * setup git to commit and push to branch main
 * replace the sources folders in `main` by the versions in `develop`under `main` and `internal/controller`
 * commit an push to `main`

## Developing locally

To develop/debug locally, install a go compiler/IDE of your choice, start a Kubernetes cluster and apply the helm chart:

```
helm repo add k-pipe https://k-pipe.github.io/pipeline-operator/
helm install k-pipe k-pipe/tdset-controller -n tdset
```

Checkout the branch `generated` of this repo and run the [go main file](https://github.com/k-pipe/pipeline-operator/blob/generated/cmd/main.go).

You can then apply some of the [test resources](https://github.com/k-pipe/pipeline-operator/tree/generated/main/tests) and observe the
resulting actions logged to the console.

## Further material
|                                           |                                                                                          |
|-------------------------------------------|------------------------------------------------------------------------------------------|
| Tutorial for building go based operators  | https://sdk.operatorframework.io/docs/building-operators/helm/quickstart/                |
| Nice introduction to kubernetes operators | https://shahin-mahmud.medium.com/write-your-first-kubernetes-operator-in-go-177047337eae |
| Another medium article that was helpful   | https://www.faizanbashir.me/guide-to-create-kubernetes-operator-with-golang              |
