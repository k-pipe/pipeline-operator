echo ""
echo "============="
echo "Installing go"
echo "============="
echo ""
wget -nv https://go.dev/dl/go1.22.2.linux-amd64.tar.gz
sudo tar -xvf go1.22.2.linux-amd64.tar.gz > /dev/null
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
ls -l $GOBIN
make install
cd ..
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
make bundle IMG="kpipe/nginx-operator:v0.0.8"
#RUN make bundle-build bundle-push IMG="kpipe/nginx-operator:v0.0.1"
