// cd $GOPATH/src/github.com/DTFN/dtfn/cmd/dtfn
if [ -f "/usr/bin/dtfn" ]; then
        rm -r /usr/bin/dtfn
fi
go build
mv dtfn /usr/bin
