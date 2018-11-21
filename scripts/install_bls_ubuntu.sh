#!/usr/bin/env bash

installBLS(){
    sudo apt-get install -y libgmp-dev libssl-dev openssl gcc
    mkdir -p $GOPATH/src/github.com/herumi

    cd $GOPATH/src/github.com/herumi
    git clone https://github.com/green-element-chain/mcl.git && cd $GOPATH/src/github.com/herumi/mcl
    git reset --hard fe95b63cc450bc1eb0459dda916a858b5442a258 && make

    cd $GOPATH/src/github.com/herumi
    git clone https://github.com/green-element-chain/bls.git && cd $GOPATH/src/github.com/herumi/bls
    git reset --hard e95a58e9d6e83f940cf6fec5906c4599ef455282 && make
}

installBLS
