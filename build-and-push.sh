echo ""
echo "============="
echo "Installing go"
echo "============="
echo ""
sudo apt-get update
sudo apt-get upgrade
sudo apt-get install golang
go version
which go
ls -l /usr/bin/go
export GOBIN=/usr/local/go/bin
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
sudo make install
echo ""
echo "====================="
echo "Creating api         "
echo "====================="
echo ""

# https://docs.docker.com/engine/install/ubuntu/ \
#RUN apt-get install -y sudo \
# && curl -fsSL https://get.docker.com -o get-docker.sh \
# && sudo sh ./get-docker.sh
operator-sdk init --domain kpipe --plugins helm
operator-sdk create api --group demo --version v1alpha1 --kind Nginx \
#RUN make bundle IMG="kpipe/nginx-operator:v0.0.1"  # give some input here
#RUN make bundle-build bundle-push IMG="kpipe/nginx-operator:v0.0.1"
