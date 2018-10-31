#!/usr/bin/env bash
installEthereum(){
    mkdir -p $GOPATH/src/github.com/ethereum  && cd $_
    git clone git@github.com:green-element-chain/go-ethereum.git
}


installTendermint(){
    cd ../tendermint/
    git clone git@github.com:green-element-chain/tendermint.git
    cd tendermint
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
}


modifyDepVersion(){
    cd $GOPATH/src/github.com/cosmos/cosmos-sdk
    git reset --hard 1e26ba2e0e9c1e0457383ff302a97396c227cddb
    cd $GOPATH/src/github.com/tendermint/go-amino
    git reset --hard c344a05f4a36122a04508bee480ada33dc8be350
}


yum install git curl wget jq gcc make -y
curl https://glide.sh/get | sh
installEthereum
installTendermint
installEthermintDependency
modifyDepVersion