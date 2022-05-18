#!/usr/bin/env bash
export GOMODCACHE=/home/duanlian/go/pkg/mod

installBLS(){
    mkdir -p $GOMODCACHE/github.com/herumi
    cd $GOMODCACHE/github.com/herumi
    if [ -d mcl ]
        then
                cd mcl
                git pull
        else
        git clone https://github.com/DTFN/mcl.git && cd $GOMODCACHE/github.com/herumi/mcl
    fi
    git reset --hard 5fd1dc64ef2ef04014bfadcb3c2ad0c54edf794b && make
    mv lib/libmcl.so lib/libmcl_dy.so
    cd $GOMODCACHE/github.com/herumi
    if [ -d bls ]
        then
                cd bls
                git pull
        else
        git clone https://github.com/DTFN/bls.git && cd $GOMODCACHE/github.com/herumi/bls
    fi
    git reset --hard f53dadd5a51900f94b7aecff0063feada2f4bb30 && make
    mv lib/libbls384.so lib/libbls384_dy.so
    mv lib/libbls256.so lib/libbls256_dy.so
    mv lib/libbls384_256.so lib/libbls384_256_dy.so
}

installUSB(){
    mkdir -p $GOMODCACHE/github.com/karalabe
    cd $GOMODCACHE/github.com/karalabe
    if [ -d usb ]
        then
                cd usb
                git pull
        else
        git clone https://github.com/karalabe/usb.git && cd $GOMODCACHE/github.com/karalabe/usb
    fi
    git reset --hard 911d15fe12a9c411cf5d0dd5635231c759399bed
}


installBLS
installUSB
