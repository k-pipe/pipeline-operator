GO_VERSION=1.21.9
BUNDLE_VERSION=v0.1.0
DOMAIN=kpipe
REPO=github.com/k-pipe/operator
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
make install
cd ..
echo ""
echo "====================="
echo "Init operator        "
echo "====================="
echo ""
mkdir operator
cd operator
operator-sdk init --domain $DOMAIN --repo $REPO --plugins=go.kubebuilder.io/v4
echo ""
echo "====================="
echo "Creating api         "
echo "====================="
echo ""
operator-sdk create api --group schedule --version v1 --kind TDSet --resource --controller
echo ""
echo "====================="
echo "Adding api sources   "
echo "====================="
echo ""
cp ../develop-locally/source/api/* api/v1/
ls -l api/v1
echo ""
echo "====================="
echo "Generating manifests "
echo "====================="
echo ""
make generate
make manifests
echo ""
echo "=========================="
echo "Commit crds to helm branch"
echo "=========================="
ls -lRt
git config remote.origin.fetch '+refs/heads/*:refs/remotes/origin/*'
echo Unsahllow
git fetch --unshallow
#echo Fetchall
#git fetch --all
echo Checkout
git checkout helm -
#-track remote/helm
#echo Checkout
#git pull
ls -lRt
cp config/crd/bases/*.yaml charts/tdset/crds/
git add charts/tdset/crds/
git commit -m "added crds"
git push
git checkout main
echo ""
make generate
make manifests
echo ""
echo "=========================="
echo "Adding controller sources "
echo "=========================="
echo ""
cp ../develop-locally/source/controller/* internal/controller
ls -l internal/controller
echo ""
echo "====================="
echo "Building             "
echo "====================="
echo ""
make build
#echo ""
#echo "====================="
#echo "Creating bundle      "
#echo "====================="
#echo ""
#mkdir -p config/manifests/bases
#cp ../operator.clusterserviceversion.yaml config/manifests/bases/
#ls -l config/manifests/bases/
#operator-sdk  generate kustomize manifests --interactive=false
#ls -l config/manifests/bases/
#make bundle IMG="kpipe/pipeline-operator:$BUNDLE_VERSION"
echo "============================"
echo "  Logging in to dockerhub"
echo "============================"
docker login --username kpipe --password $DOCKERHUB_PUSH_TOKEN
echo ""
echo "====================="
echo "Build & Push image  "
echo "====================="
echo ""
make docker-build docker-push IMG="kpipe/pipeline-operator:$BUNDLE_VERSION"
#echo ""
#echo "====================="
#echo "Build & Push bundle  "
#echo "====================="
#echo ""
#make bundle-build bundle-push IMG="kpipe/pipeline-operator:$BUNDLE_VERSION"
#echo ""
#echo "======================="
#echo "Deploy to test-cluster "
#echo "======================="
#echo ""
#echo $SERVICE_ACCOUNT_JSON_KEY > key.json
#echo "Json key has" `grep -c "" key.json` "lines"
#gcloud auth activate-service-account github-ci-cd@k-pipe-test-system.iam.gserviceaccount.com --key-file=key.json --project=k-pipe-test-system
#echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
#curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
#sudo apt update
#sudo apt-get install google-cloud-sdk-gke-gcloud-auth-plugin kubectl
#export USE_GKE_GCLOUD_AUTH_PLUGIN=True
#gcloud  container clusters get-credentials k-pipe-runner --region europe-west3
#make deploy IMG="kpipe/operator-bundle:$BUNDLE_VERSION"


