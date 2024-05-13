GO_VERSION=1.21.9
DOMAIN=k-pipe.cloud
DOCKER_USER=kpipe
GITHUB_USER=k-pipe
APP_NAME=pipeline-operator
GROUP=pipeline
API_VERSION=v1
REPO=github.com/$GITHUB_USER/$APP_NAME
KUBEBUILDER_PLUGIN=go.kubebuilder.io/v4
KIND=TDSet
LC_KIND=tdset
#
echo ""
echo "==========================="
echo "Configuring git repo access"
echo "==========================="
echo ""
git fetch origin helm:helm --force
git fetch origin generated:generated --force
git remote set-url origin https://$GITHUB_USER:$CICD_GITHUB_TOKEN@$REPO.git
git config --global user.email "$GITHUB_USER@$DOMAIN"
git config --global user.name "$GITHUB_USER"
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
# do installation in tmp folder
#
mkdir tmp
cd tmp
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
cd ../..
rm -rf tmp
echo ""
echo "====================="
echo "Init operator        "
echo "====================="
echo ""
mkdir operator
cd operator
operator-sdk init --domain $DOMAIN --repo $REPO --plugins=$KUBEBUILDER_PLUGIN
echo ""
echo "====================="
echo "Creating api         "
echo "====================="
echo ""
operator-sdk create api --group $GROUP --version $API_VERSION --kind $KIND --resource --controller
echo ""
echo "====================="
echo "Adding api sources   "
echo "====================="
echo ""
cp ../source/api/* api/$API_VERSION/
ls -l api/$API_VERSION
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
rm ../charts/$LC_KIND/crds/*.yaml
cp ../operator/config/crd/bases/*.yaml ../charts/$LC_KIND/crds/
ls -l ../charts/$LC_KIND/crds/
git add ../charts/$LC_KIND/crds
echo $VERSION > ../version
git add ../version
sed -i "s#version: .*#version: $VERSION#" ../charts/$LC_KIND/Chart.yaml
sed -i "s#appVersion: .*#appVersion: $VERSION#" ../charts/$LC_KIND/Chart.yaml
git add ../charts/$LC_KIND/Chart.yaml
sed -i "s#version .*#version $VERSION#" ../charts/$LC_KIND/templates/NOTES.txt
git add ../charts/$LC_KIND/templates/NOTES.txt
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
docker login --username $DOCKER_USER --password $DOCKERHUB_PUSH_TOKEN
echo ""
echo "====================="
echo "Build & Push image  "
echo "====================="
echo ""
make docker-build docker-push IMG="$DOCKER_USER/$APP_NAME:$VERSION"
echo ""
echo "========================="
echo "Push to branch generated "
echo "========================="
echo ""
cd ..
git checkout generated
ls | grep -xv "operator" | sed "s#^#rm -f #" | sh
mv operator/* .
mv operator/.* .
rm -f operator
echo $VERSION > version
git add .
git commit -m "version $VERSION"
git push --set-upstream origin generated
