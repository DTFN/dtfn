#!/usr/bin/env bash
installEthereum(){
    mkdir -p $GOPATH/src/github.com/ethereum  && cd $_
    git clone git@github.com:green-element-chain/go-ethereum.git
    cd go-ethereum
    git checkout ForbidNormalPeer
    go mod vendor
    cd cmd/geth && go install
    cd ../../../..
}


installTendermint(){
    mkdir tendermint && cd tendermint/
    git clone git@github.com:green-element-chain/tendermint.git
    cd tendermint
    #git checkout -b develop remotes/origin/develop
    git checkout ForbidNormalPeer
    go mod vendor
    cd cmd/tendermint && go install
    cd ../../../..
}

installGelchain(){
    cd green-element-chain/gelchain
    #git checkout -b develop remotes/origin/develop
    git checkout ForbidNormalPeer
    go mod vendor
    backupVendors
    export GO111MODULE=off
    cd cmd/gelchain && go install
    cd ../..
    restoreVendors
    export GO111MODULE=on
}

backupVendors(){
    mkdir vendor_bak
    mv ../../ethereum/go-etherem/vendor/github.com/tendermint/tendermint vendor_bak/ethereum/go-etherem/vendor/github.com/tendermint/tendermint
    mv vendor/github.com/tendermint/tendermint vendor_bak/green-element-chain/gelchain/vendor/github.com/tendermint/tendermint
    mv ../../tendermint/tendermint/vendor/github.com/ethereum/go-ethereum vendor_bak/tendermint/tendermint/vendor/github.com/ethereum/go-ethereum
    mv vendor/github.com/ethereum/go-ethereum vendor_bak/green-element-chain/gelchain/vendor/github.com/ethereum/go-ethereum
    rm ../../ethereum/go-etherem/vendor/github.com/karalabe
    rm ../../ethereum/go-etherem/vendor/gopkg.in -rf
    rm ../../../gopkg.in -rf
    mv vendor/gopkg.in ../../.. -rf
}


restoreVendors(){
    mv vendor_bak/ethereum/go-etherem/vendor/github.com/tendermint/tendermint ../../ethereum/go-etherem/vendor/github.com/tendermint
    mv vendor_bak/green-element-chain/gelchain/vendor/github.com/tendermint/tendermint vendor/github.com/tendermint
    mv vendor_bak/tendermint/tendermint/vendor/github.com/ethereum/go-ethereum ../../tendermint/tendermint/vendor/github.com/ethereum
    mv vendor_bak/green-element-chain/gelchain/vendor/github.com/ethereum/go-ethereum vendor/github.com/ethereum
    cp ../../karalabe ../../ethereum/go-etherem/vendor/github.com/ -rf
    cp ../../../gopkg.in vendor ../../ethereum/go-etherem/vendor -rf
}

export GO111MODULE=on
installEthereum
installTendermint
installGelchain
