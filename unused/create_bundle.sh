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


