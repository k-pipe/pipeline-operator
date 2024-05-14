#
# Script to push changes to branch "helm". Replaces the version in various chart files and commits them to the helm
# branch together with CRD definitions expected in folder operator/config/crd/bases/
#
echo ""
echo "=============================="
echo "Commit and push to helm branch"
echo "=============================="
VERSION=`cat version`
echo "Version found in main: $VERSION"
#
# switch to helm branch
#
git checkout helm
#
# copy CRD definitions
#
rm charts/$LC_KIND/crds/*.yaml
cp operator/config/crd/bases/*.yaml charts/$LC_KIND/crds/
echo ""
echo "Contents of folder with CRDs:"
ls -l charts/$LC_KIND/crds/
echo ""
git add charts/$LC_KIND/crds
#
# update version file
#
echo $VERSION > version
git add ../version
#
# replace version in various files
#
sed -i "s#version: .*#version: $VERSION#" ../charts/$LC_KIND/Chart.yaml
sed -i "s#appVersion: .*#appVersion: $VERSION#" ../charts/$LC_KIND/Chart.yaml
git add ../charts/$LC_KIND/Chart.yaml
sed -i "s#version .*#version $VERSION#" ../charts/$LC_KIND/templates/NOTES.txt
git add ../charts/$LC_KIND/templates/NOTES.txt
#
# commit and push
#
git commit --allow-empty -m "version $VERSION"
git push --set-upstream origin helm
#
# switch back to main branch
#
git checkout main
