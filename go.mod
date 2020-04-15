module github.com/green-element-chain/gelchain

        go 1.14

        require (
        github.com/cosmos/cosmos-sdk v0.38.3
        github.com/ethereum/go-ethereum v1.9.12
        github.com/golang/mock v1.4.3
        github.com/mattn/go-colorable v0.1.6
        github.com/stretchr/testify v1.5.1
        github.com/tendermint/tendermint v0.33.3
        gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951 // indirect
        gopkg.in/urfave/cli.v1 v1.20.0
        )

        replace github.com/ethereum/go-ethereum => github.com/green-element-chain/go-ethereum v1.4.6

        replace github.com/tendermint/tendermint => github.com/green-element-chain/tendermint v1.4.6