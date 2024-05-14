GO_VERSION=1.21.9
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
echo ""
echo "Go version:"
go version
echo ""
echo "Go executable:"
which go
cd ..
rm -rf tmp
