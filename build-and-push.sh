GO_VERSION=1.21.9
DOMAIN=k-pipe.cloud
DOCKER_USER=kpipe
GITHUB_USER=k-pipe
APP_NAME=pipeline-operator
GROUP=pipeline
API_VERSION=v1
REPO=github.com/$GITHUB_USER/$APP_NAME
KUBEBUILDER_PLUGIN=go.kubebuilder.io/v4
CHART=pipeline
#
echo ""
echo "==========================="
echo "Configuring git repo access"
echo "==========================="
echo ""
git fetch origin helm:helm --force
git fetch origin generated:generated --force
git remote set-url origin https://cicd-k-pipe:$CICD_GITHUB_TOKEN@$REPO.git
git config --global user.email "cicd@k-pipe.cloud"
git config --global user.name "cicd-k-pipe"
#
VERSION=`cat version`
echo "Tagging version: $VERSION"
git tag $VERSION
git push origin $VERSION
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
if [ $? != 0 ]
then
  echo Init operator failed
  exit 1
fi
echo ""
echo "====================="
echo "Creating apis        "
echo "====================="
echo ""
APIDIR=../source/api
APIS=`ls -1 $APIDIR`
for APISOURCE in $APIS
do
   KIND=`cat $APIDIR/$APISOURCE | grep "^type" | tail -2 | head -1 | awk '{print $2}'`
   echo "Creating kind $KIND (source: $APISOURCE)"
   operator-sdk create api --group $GROUP --version $API_VERSION --kind $KIND --resource --controller
   if [ $? != 0 ]
   then
     echo Creating API failed
     exit 1
   fi
done
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
if [ $? != 0 ]
then
  echo Generating manifests failed
  exit 1
fi
echo ""
echo "=========================="
echo "Commit crds to helm branch"
echo "=========================="
git checkout helm
rm ../charts/$CHART/crds/*.yaml
cp ../operator/config/crd/bases/*.yaml ../charts/$CHART/crds/
ls -l ../charts/$CHART/crds/
git add ../charts/$CHART/crds
echo $VERSION > ../version
git add ../version
sed -i "s#version: .*#version: $VERSION#" ../charts/$CHART/Chart.yaml
sed -i "s#appVersion: .*#appVersion: $VERSION#" ../charts/$CHART/Chart.yaml
git add ../charts/$CHART/Chart.yaml
sed -i "s#version .*#version $VERSION#" ../charts/$CHART/templates/NOTES.txt
git add ../charts/$CHART/templates/NOTES.txt
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
if [ $? != 0 ]
then
  echo Build failed
  exit 1
fi
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
# move folder tests to be copied to generated branch
mkdir operator/main
mv source/tests operator/main
# move folder api to be also copied to generated branch
mv source/api operator/main
# go to branch "generated", this will keep the folder "operator" since it is not checked in into main
git checkout generated
# remove all files except operator folder
ls | grep -xv "operator" | grep -xv "." | grep -xv ".." | sed "s#^#rm -rf #" | sh
# move files from operator folder
mv operator/* .
# delete operator folder
rm -rf operator
# add version
echo $VERSION > version
# add all files to git
git add -A .
# commit and push
git commit -m "version $VERSION"
git push --set-upstream origin generated
