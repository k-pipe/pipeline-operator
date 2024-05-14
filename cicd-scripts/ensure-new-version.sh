#
# Checking if version in main branch is same as in branch helm, if so increase final number
#
echo ""
echo "================"
echo "Checking version"
echo "================"
echo ""
CURRENT_VERSION=`cat version`
echo "Version in branch main: $CURRENT_VERSION"
echo "Version in branch helm: $HELM_VERSION"
if [[ $CURRENT_VERSION = $HELM_VERSION ]]
then
  echo ""
  echo "===================="
  echo "Incrementing version"
  echo "===================="
  echo ""
  NEW_VERSION=`echo $CURRENT_VERSION | sed 's#[0-9]*$##'``echo $CURRENT_VERSION+1 | sed "s#^.*\.##" | bc -l`
  echo "New version: $NEW_VERSION"
  echo $NEW_VERSION > version
fi
