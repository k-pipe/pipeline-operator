FROM golang:1.20
#
# Install kubectl & auth plugin
#
RUN apt-get update
RUN apt-get install sudo
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
 && apt-get install -y gnupg \
 && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - \
 && sudo apt update \
 && sudo apt-get install -y google-cloud-cli google-cloud-sdk-gke-gcloud-auth-plugin kubectl
ENV USE_GKE_GCLOUD_AUTH_PLUGIN=True
COPY --from=kpipe/pipeline-operator:v0.1.0 /manager /manager
ENTRYPOINT /bin/sh


