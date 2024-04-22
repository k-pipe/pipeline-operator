GO_VERSION=1.22.2
BUNDLE_VERSION=v0.0.9
echo ""
echo "============="
echo "Installing go"
echo "============="
echo ""
echo "Go version:" $GO_VERSION
wget -nv https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz
sudo tar -xvf go$GO_VERSION.linux-amd64.tar.gz > /dev/null
sudo mv go /usr/local
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOBIN=/usr/local/go/bin/
go version
which go
echo ""
echo "===================="
echo "Checking out sources"
echo "===================="
echo ""
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
echo ""
echo "====================="
echo "Building operator-sdk"
echo "====================="
echo ""
sudo chmod a+w+x $GOBIN
ls -l $GOBIN
make install
cd ..
echo ""
echo "====================="
echo "Init operator        "
echo "====================="
echo ""
mkdir operator
cd operator
operator-sdk init --domain kpipe --plugins helm
echo ""
echo "====================="
echo "Creating api         "
echo "====================="
echo ""
operator-sdk create api --group demo --version v1alpha1 --kind Nginx
echo ""
echo "====================="
echo "Creating bundle      "
echo "====================="
echo ""
mkdir -p config/manifests/bases
cp ../operator.clusterserviceversion.yaml config/manifests/bases/
ls -l config/manifests/bases/
#operator-sdk  generate kustomize manifests --interactive=false
#ls -l config/manifests/bases/
make bundle IMG="kpipe/nginx-operator:$BUNDLE_VERSION"
echo "============================"
echo "  Logging in to dockerhub"
echo "============================"
docker login --username kpipe --password $DOCKERHUB_PUSH_TOKEN
echo ""
echo "====================="
echo "Build & Push bundle  "
echo "====================="
echo ""
make bundle-build bundle-push IMG="kpipe/nginx-operator:$BUNDLE_VERSION"
echo ""
echo "======================="
echo "Deploy to test-cluster "
echo "======================="
echo ""
echo $SERVICE_ACCOUNT_JSON_KEY
echo $SERVICE_ACCOUNT_JSON_KEY > key.json
grep -c "" key.json
echo Key:
cat key.json
gcloud auth activate-service-account github-ci-cd@k-pipe-test-system.iam.gserviceaccount.com --key-file=key.json --project=k-pipe-test-system
gcloud  container clusters get-credentials k-pipe-runner --region europe-west3
make deploy IMG="kpipe/operator-bundle:$BUNDLE_VERSION"

