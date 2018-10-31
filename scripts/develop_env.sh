#!/usr/bin/env bash
installEthereum(){
    mkdir -p $GOPATH/src/github.com/ethereum  && cd $_
    git clone git@github.com:green-element-chain/go-ethereum.git
}


installTendermint(){
    cd ../tendermint/
    mkdir tendermint && cd tendermint
    git clone git@github.com:green-element-chain/tendermint.git
    cd tendermint
    make get_tools
    make get_vendor_deps
    rm -r vendor/github.com/ethereum
    make install
}


installEthermintDependency(){
    go get github.com/cosmos/cosmos-sdk
    go get github.com/mattn/go-colorable
    go get gopkg.in/urfave/cli.v1
    go get github.com/golang/protobuf/proto
    go get github.com/tendermint/go-amino
}


modifyDepVersion(){
    cd $GOPATH/src/github.com/cosmos/cosmos-sdk
    git reset --hard v0.23.1
    cd $GOPATH/src/github.com/tendermint/go-amino
    git reset --hard v0.12.0
}


apt-get install git curl wget jq gcc make -y
curl https://glide.sh/get | sh
installEthereum
installTendermint
installEthermintDependency
modifyDepVersion