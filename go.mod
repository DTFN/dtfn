module github.com/DTFN/gelchain

go 1.14

require (
	bazil.org/fuse v0.0.0-20200415052832-70bd89b671a2 // indirect
	github.com/cosmos/cosmos-sdk v0.38.3 // indirect
	github.com/ethereum/go-ethereum v1.9.12
	github.com/golang/mock v1.4.3
	github.com/karalabe/hid v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.6
	github.com/robertkrimen/otto v0.0.0-20191219234010-c382bd3c16ff // indirect
	github.com/stretchr/testify v1.5.1
	github.com/tendermint/tendermint v0.33.3
	gopkg.in/fatih/set.v0 v0.2.1 // indirect
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/ethereum/go-ethereum => github.com/DTFN/go-ethereum v1.5.0

replace github.com/tendermint/tendermint => github.com/DTFN/tendermint v1.5.0
