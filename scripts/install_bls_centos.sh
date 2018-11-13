#!/usr/bin/env bash

installBLS(){
    sudo yum install -y gmp-devel openssl-devel gcc
    mkdir -p $GOPATH/src/github.com/herumi

    cd $GOPATH/src/github.com/herumi
    git clone https://github.com/green-element-chain/mcl.git && cd $GOPATH/src/github.com/herumi/mcl
    git reset --hard fe95b63cc450bc1eb0459dda916a858b5442a258 && make

    cd $GOPATH/src/github.com/herumi
    git clone https://github.com/green-element-chain/bls.git && cd $GOPATH/src/github.com/herumi/bls
    git reset --hard e9c72f18ab9bc09923da739151821cc588c0d295 && make
}

installBLS
