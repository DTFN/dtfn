cd $GOPATH/src/github.com/green-element-chain/gelchain/cmd/gelchain
rm -r /usr/bin/gelchain
go build
mv gelchain /usr/bin
