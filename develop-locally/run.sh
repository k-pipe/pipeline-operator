rm -rf operator
docker run -it --platform linux/amd64 \
 -v $PWD/operator:/operator \
 -v ~/.config/gcloud:/root/.config/gcloud \
 controller-dev

# -v /var/run/docker.sock:/var/run/docker.sock -v ~/.config/gcloud:/root/.config/gcloud test
