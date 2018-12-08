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
    git checkout geltm-pos
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
}


modifyDepVersion(){
    cd $GOPATH/src/github.com/cosmos/cosmos-sdk
    git reset --hard aa72e72ce4bf8dcd3a9897f32010c4dae9a376f7
    cd $GOPATH/src/github.com/tendermint/go-amino
    git reset --hard 833d32f0923bc8272155560b3ee602f05d579d75
}


yum install git curl wget jq gcc make -y
curl https://glide.sh/get | sh
installEthereum
installTendermint
installEthermintDependency
modifyDepVersion