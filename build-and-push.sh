export GOBIN=/usr/local/go/bin
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
echo "Installing"
apt-get update
apt-get upgrade
apt-get install go-lang
ls -l /usr/local/go
chmod u+x /usr/local/go
sudo make install
# https://docs.docker.com/engine/install/ubuntu/ \
#RUN apt-get install -y sudo \
# && curl -fsSL https://get.docker.com -o get-docker.sh \
# && sudo sh ./get-docker.sh
operator-sdk init --domain kpipe --plugins helm
operator-sdk create api --group demo --version v1alpha1 --kind Nginx \
#RUN make bundle IMG="kpipe/nginx-operator:v0.0.1"  # give some input here
#RUN make bundle-build bundle-push IMG="kpipe/nginx-operator:v0.0.1"
