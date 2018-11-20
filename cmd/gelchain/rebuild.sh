cd $GOPATH/src/github.com/green-element-chain/gelchain/cmd/gelchain
if [ -f "/usr/bin/gelchain" ]; then
        rm -r /usr/bin/gelchain
fi
go build
mv gelchain /usr/bin
