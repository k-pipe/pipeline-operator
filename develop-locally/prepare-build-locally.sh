# install operator sdk and build operator in a docker image
docker build . --platform linux/amd64 -t controller-dev

# get cluster credentials and run operator
docker run -it --platform linux/amd64 \
 -v ~/.config/gcloud:/root/.config/gcloud \
 -v $PWD/local:/local \
 controller-dev \
 cp -r /operator /local

