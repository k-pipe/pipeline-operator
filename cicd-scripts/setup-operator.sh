DOMAIN=k-pipe.cloud
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
cd..
