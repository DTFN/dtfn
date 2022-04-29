module github.com/DTFN/dtfn

go 1.13

require (
	github.com/ethereum/go-ethereum v1.9.12
	github.com/golang/mock v1.1.1
	github.com/mattn/go-colorable v0.1.0
	github.com/stretchr/testify v1.5.1
	github.com/tendermint/tendermint v0.33.3
	gopkg.in/urfave/cli.v1 v1.20.0
	gopkg.in/yaml.v2 v2.2.5
)

replace (
	github.com/ethereum/go-ethereum => github.com/DTFN/go-ethereum v1.5.2
	github.com/tendermint/tendermint => github.com/DTFN/tendermint v1.5.0
)
