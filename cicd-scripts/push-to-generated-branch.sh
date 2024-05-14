GO_VERSION=1.21.9
DOMAIN=k-pipe.cloud
DOCKER_USER=kpipe
GITHUB_USER=k-pipe
APP_NAME=pipeline-operator
GROUP=pipeline
API_VERSION=v1
REPO=github.com/$GITHUB_USER/$APP_NAME
KUBEBUILDER_PLUGIN=go.kubebuilder.io/v4
KIND=TDSet
LC_KIND=tdset
#
echo ""
echo "==========================="
echo "Configuring git repo access"
echo "==========================="
echo ""
git fetch origin helm:helm --force
git fetch origin generated:generated --force
git remote set-url origin https://$GITHUB_USER:$CICD_GITHUB_TOKEN@$REPO.git
git config --global user.email "$GITHUB_USER@$DOMAIN"
git config --global user.name "$GITHUB_USER"

echo ""
echo "========================="
echo "Push to branch generated "
echo "========================="
echo ""
cd ..
# move folder tests
mv tests operator
# go to branch "generated", this will keep the folder "operator" since it is not checked in into main
git checkout generated
# remove all files except operator folder
ls | grep -xv "operator" | grep -xv "." | grep -xv ".." | sed "s#^#rm -rf #" | sh
# move files from operator folder
mv operator/* .
# delete operator folder
rm -rf operator
# addd version
echo $VERSION > version
# add all files to git
git add -A .
# commit and push
git commit -m "version $VERSION"
git push --set-upstream origin generated
