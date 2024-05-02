cd /operator
#DOMAIN=k-pipe.cloud
DOMAIN=rs
operator-sdk init --domain $DOMAIN --repo github.com/BackAged/tdset-operator --plugins=go.kubebuilder.io/v4
operator-sdk create api --group schedule --version v1 --kind TDSet --resource --controller
cp /source/api/tdset_types.go api/v1/
make generate
make manifests
cp /source/controller/* internal/controller/
make build
