cd $GOPATH/src/github.com/DTFN/gelchain/cmd/gelchain
if [ -f "/usr/bin/gelchain" ]; then
        rm -r /usr/bin/gelchain
fi
go build
mv gelchain /usr/bin
