# Build the manager binary
FROM golang:1.21
# see https://sdk.operatorframework.io/docs/installation/
#RUN apt-get update  \
#  && apt-get upgrade -y  \
#  && apt-get install -y git make golang-go
ENV GOBIN /usr/local/go/bin
RUN git clone https://github.com/operator-framework/operator-sdk \
 && cd operator-sdk \
 && git checkout master \
 && make install
# see https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/
RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
 && chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
 && echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list \
 && chmod 644 /etc/apt/sources.list.d/kubernetes.list \
 && apt-get update \
 && apt-get install -y kubectl

# https://docs.docker.com/engine/install/ubuntu/ \
#RUN apt-get install -y sudo \
# && curl -fsSL https://get.docker.com -o get-docker.sh \
# && sudo sh ./get-docker.sh
# https://minikube.sigs.k8s.io/docs/start/
#RUN curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
# && install minikube-linux-amd64 /usr/local/bin/minikube \
# && rm minikube-linux-amd64
# see https://sdk.operatorframework.io/docs/building-operators/helm/quickstart/ \
RUN operator-sdk init --domain kpipe --plugins helm
RUN operator-sdk create api --group demo --version v1alpha1 --kind Nginx \
#RUN make docker-build docker-push IMG="kpipe/nginx-operator:v0.0.1"
ADD info.txt .

# copy gcloud cli to target image
#COPY --from=gcr.io/google.com/cloudsdktool/google-cloud-cli:432.0.0 /usr/lib/google-cloud-sdk /usr/lib/google-cloud-sdk
#ENV PATH $PATH:/usr/lib/google-cloud-sdk/bin/
#RUN gcloud  container clusters get-credentials k-pipe-runner --region europe-west3

#RUN operator-sdk olm install \
#RUN make bundle IMG="kpipe/nginx-operator:v0.0.1"  # give some input here
#RUN make bundle-build bundle-push IMG="kpipe/nginx-operator:v0.0.1"
