docker build . -t test
docker run -it -v ~/.config/gcloud:/root/.config/gcloud test
#-v ~/.kube/config:/kube/config --env KUBECONFIG=/kube/config test # /manager

