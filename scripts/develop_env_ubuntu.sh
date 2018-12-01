#!/usr/bin/env bash
installEthereum(){
    mkdir -p $GOPATH/src/github.com/ethereum  && cd $_
    git clone git@github.com:green-element-chain/go-ethereum.git
}


installTendermint(){
    cd ../
    mkdir tendermint && cd tendermint/
    git clone git@github.com:green-element-chain/tendermint.git
    cd tendermint
    git checkout bls
    make get_tools
    make get_vendor_deps
    rm -r vendor/github.com/ethereum
    rm -r vendor/github.com/tendermint/go-amino
}


installEthermintDependency(){
    go get github.com/cosmos/cosmos-sdk
    go get github.com/mattn/go-colorable
    go get gopkg.in/urfave/cli.v1
    go get github.com/golang/protobuf/proto
    go get github.com/tendermint/go-amino
    go get github.com/stretchr/testify/require
    go get github.com/spaolacci/murmur3
    go get github.com/golang/mock/gomock
    go get github.com/rs/cors
    go get github.com/tendermint/btcd
    cd $GOPATH/src/github.com/green-element-chain/gelchain
    git checkout bls
}


modifyDepVersion(){
    cd $GOPATH/src/github.com/cosmos/cosmos-sdk
    git reset --hard 1e26ba2e0e9c1e0457383ff302a97396c227cddb
    cd $GOPATH/src/github.com/tendermint/go-amino
    git reset --hard dc14acf9ef15f85828bfbc561ed9dd9d2a284885
}


apt-get install git curl wget jq gcc make -y
curl https://glide.sh/get | sh
installEthereum
installTendermint
installEthermintDependency
modifyDepVersion
