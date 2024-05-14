#
echo ""
echo "===================="
echo "Checking out sources"
echo "===================="
echo ""
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
echo ""
echo "====================="
echo "Building operator-sdk"
echo "====================="
echo ""
sudo chmod a+w+x $GOBIN
make install
cd ..
rm -rf operator-sdk

