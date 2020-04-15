#!/usr/bin/env bash

installBLS(){
    #sudo apt-get install -y libgmp-dev libssl-dev openssl gcc g++
    apt-get install -y libgmp-dev libssl-dev openssl gcc g++
    mkdir -p $GOPATH/src/github.com/herumi

    cd $GOPATH/src/github.com/herumi
    if [ -d mcl ]
        then
                cd mcl
                git pull
        else
        git clone https://github.com/green-element-chain/mcl.git && cd $GOPATH/src/github.com/herumi/mcl
    fi
    git reset --hard fe95b63cc450bc1eb0459dda916a858b5442a258 && make
    cd $GOPATH/src/github.com/herumi
    if [ -d bls ]
        then
                cd bls
                git pull
        else
        git clone https://github.com/green-element-chain/bls.git && cd $GOPATH/src/github.com/herumi/bls
    fi
    git reset --hard e9c72f18ab9bc09923da739151821cc588c0d295 && make
}

installUSB(){
    mkdir -p $GOPATH/src/github.com/karalabe

    cd $GOPATH/src/github.com/karalabe
    if [ -d usb ]
        then
                cd usb
                git pull
        else
        git clone https://github.com/karalabe/usb.git && cd $GOPATH/src/github.com/karalabe/usb
    fi
    git reset --hard 911d15fe12a9c411cf5d0dd5635231c759399bed
}


installBLS
installUSB
