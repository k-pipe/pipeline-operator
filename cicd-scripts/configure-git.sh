DOMAIN=k-pipe.cloud
GITHUB_USER=k-pipe
APP_NAME=pipeline-operator
REPO=github.com/$GITHUB_USER/$APP_NAME
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
