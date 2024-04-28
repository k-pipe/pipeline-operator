# call install script
sh scripts/install.sh
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
gcloud  container clusters get-credentials k-pipe-runner --region europe-west3
make deploy IMG="kpipe/operator-bundle:$BUNDLE_VERSION"

