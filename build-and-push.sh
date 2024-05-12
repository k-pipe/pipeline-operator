GO_VERSION=1.21.9
#IMAGE_VERSION=0.0.1
#BUNDLE_VERSION=v0.1.0
DOMAIN=kpipe
REPO=github.com/k-pipe/pipeline-operator
#
echo ""
echo "==========================="
echo "Configuring git repo access"
echo "==========================="
echo ""
git fetch origin helm:helm --force
git remote set-url origin https://k-pipe:$CICD_GITHUB_TOKEN@github.com/k-pipe/pipeline-operator.git
git config --global user.email "k-pipe@kneissler.com"
git config --global user.name "k-pipe"
#
echo ""
echo "=================="
echo "Increasing version"
echo "=================="
echo ""
PREVIOUS_VERSION=`git show helm:version`
echo Previous version: $PREVIOUS_VERSION
VERSION=`echo $PREVIOUS_VERSION | sed 's#[0-9]*$##'``echo $PREVIOUS_VERSION+1 | sed "s#^.*\.##" | bc -l`
echo New version: $VERSION
#
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
cp ../source/api/* api/v1/
ls -l api/v1
echo ""
echo "====================="
echo "Generating manifests "
echo "====================="
echo ""
make generate manifests
echo ""
echo "=========================="
echo "Commit crds to helm branch"
echo "=========================="
git checkout helm
rm ../charts/tdset/crds/*.yaml
cp ../operator/config/crd/bases/*.yaml ../charts/tdset/crds/
ls -l ../charts/tdset/crds/
git add ../charts/tdset/crds
echo $VERSION > ../version
git add ../version
sed -i "s#version: .*#version: $VERSION#" ../charts/tdset/Chart.yaml
sed -i "s#appVersion: .*#appVersion: $VERSION#" ../charts/tdset/Chart.yaml
git add ../charts/tdset/Chart.yaml
sed -i "s#version .*#version $VERSION#" ../charts/tdset/templates/NOTES.txt
git add ../charts/tdset/templates/NOTES.txt
git commit --allow-empty -m "version $VERSION"
git push --set-upstream origin helm
git checkout main
echo ""
echo "=========================="
echo "Adding controller sources "
echo "=========================="
echo ""
cp ../source/controller/* internal/controller
ls -l internal/controller
echo ""
echo "====================="
echo "Building             "
echo "====================="
echo ""
make build
echo "============================"
echo "  Logging in to dockerhub"
echo "============================"
docker login --username kpipe --password $DOCKERHUB_PUSH_TOKEN
echo ""
echo "====================="
echo "Build & Push image  "
echo "====================="
echo ""
make docker-build docker-push IMG="kpipe/pipeline-operator:$VERSION"
