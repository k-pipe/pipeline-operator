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
echo "=========================="
echo "Adding controller sources "
echo "=========================="
echo ""
cp source/controller/* operator/internal/controller
cd operator
echo "Sources:"
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
cd..