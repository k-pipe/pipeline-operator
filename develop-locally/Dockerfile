
FROM ubuntu
ENV GO_VERSION=1.21.9
#
# install basic tools
#
RUN apt-get update && apt-get install -y wget sudo make git curl
#
# install go
#
RUN wget -nv https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz \
 && sudo tar -xvf go$GO_VERSION.linux-amd64.tar.gz > /dev/null \
 && sudo mv go /usr/local \
 && sudo chmod a+w+x /usr/local/go/bin/
ENV GOROOT=/usr/local/go
ENV GOBIN=/usr/local/go/bin/
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin
#
# build operator-sdk
#
RUN git clone https://github.com/operator-framework/operator-sdk \
 && cd operator-sdk \
 && git checkout master \
 && make install \
 && cd ..
#
# Install kubectl & auth plugin
#
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
 && apt-get install -y gnupg \
 && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - \
 && sudo apt update \
 && sudo apt-get install -y google-cloud-cli google-cloud-sdk-gke-gcloud-auth-plugin kubectl
ENV USE_GKE_GCLOUD_AUTH_PLUGIN=True
ENV DOMAIN=kpipe
ENV REPO=github.com/k-pipe/operator
RUN mkdir /operator
WORKDIR /operator
RUN operator-sdk init --domain $DOMAIN --repo $REPO --plugins=go.kubebuilder.io/v4
RUN operator-sdk create api --group schedule --version v1 --kind TDSet --resource --controller
ADD /source/api/* /operator/api/v1/
RUN make generate
RUN make manifests
ADD /source/controller/* /operator/internal/controller
RUN make build
ADD run-operator.sh /operator


