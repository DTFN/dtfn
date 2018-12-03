#!/usr/bin/env bash

installBLS(){
    apt-get install -y libgmp-dev libssl-dev openssl gcc g++
    mkdir -p $GOPATH/src/github.com/herumi

    cd $GOPATH/src/github.com/herumi
    if [ -d mcl ]
	then
		cd mcl
		git pull
	else 
    		git clone github.com/green-element-chain/mcl.git && cd $GOPATH/src/github.com/herumi/mcl
    		git reset --hard fe95b63cc450bc1eb0459dda916a858b5442a258 && make
    fi
    cd $GOPATH/src/github.com/herumi
    if [ -d bls ]
	then
		cd bls
		git pull
	else 
    		git clone https://github.com/green-element-chain/bls.git && cd $GOPATH/src/github.com/herumi/bls
    		git reset --hard e9c72f18ab9bc09923da739151821cc588c0d295 && make
    fi
}

installBLS
