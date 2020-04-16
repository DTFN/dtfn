#!/usr/bin/env bash
installEthereum(){
    if [ -d $GOPATH/src/github.com/ethereum/go-ethereum ]
        then
                cd $GOPATH/src/github.com/ethereum/go-ethereum
                git pull
        else
        	mkdir -p $GOPATH/src/github.com/ethereum  && cd $GOPATH/src/github.com/ethereum
        	pwd
        	git clone git@github.com:green-element-chain/go-ethereum.git
        	cd go-ethereum
    fi

    git checkout ForbidNormalPeer
    go mod vendor
    cd cmd/geth && go install
    cd ../../../..
}


installTendermint(){
    if [ -d $GOPATH/src/github.com/tendermint/tendermint ]
            then
                    cd $GOPATH/src/github.com/tendermint/tendermint
                    git pull
            else
            	mkdir -p $GOPATH/src/github.com/tendermint  && cd $GOPATH/src/github.com/tendermint
            	git clone git@github.com:green-element-chain/tendermint.git
            	cd tendermint
    fi
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
    mkdir -p vendor_bak/ethereum/go-ethereum/vendor/github.com/tendermint
    mv ../../ethereum/go-ethereum/vendor/github.com/tendermint/tendermint vendor_bak/ethereum/go-ethereum/vendor/github.com/tendermint
    mkdir -p vendor_bak/green-element-chain/gelchain/vendor/github.com/tendermint
    mv vendor/github.com/tendermint/tendermint vendor_bak/green-element-chain/gelchain/vendor/github.com/tendermint
    mkdir -p vendor_bak/tendermint/tendermint/vendor/github.com/ethereum
    mv ../../tendermint/tendermint/vendor/github.com/ethereum/go-ethereum vendor_bak/tendermint/tendermint/vendor/github.com/ethereum
    mkdir -p vendor_bak/green-element-chain/gelchain/vendor/github.com/ethereum
    mv vendor/github.com/ethereum/go-ethereum vendor_bak/green-element-chain/gelchain/vendor/github.com/ethereum
    rm ../../ethereum/go-ethereum/vendor/github.com/karalabe -rf
    rm ../../ethereum/go-ethereum/vendor/gopkg.in -rf
    rm ../../../gopkg.in -rf
    mv vendor/gopkg.in ../../.. -f
}


restoreVendors(){
    mv vendor_bak/ethereum/go-ethereum/vendor/github.com/tendermint/tendermint ../../ethereum/go-ethereum/vendor/github.com/tendermint
    mv vendor_bak/green-element-chain/gelchain/vendor/github.com/tendermint/tendermint vendor/github.com/tendermint
    mv vendor_bak/tendermint/tendermint/vendor/github.com/ethereum/go-ethereum ../../tendermint/tendermint/vendor/github.com/ethereum
    mv vendor_bak/green-element-chain/gelchain/vendor/github.com/ethereum/go-ethereum vendor/github.com/ethereum
    cp ../../karalabe ../../ethereum/go-ethereum/vendor/github.com/ -rf
    cp ../../../gopkg.in vendor ../../ethereum/go-ethereum/vendor -rf
    rm vendor_bak -rf
}

export GO111MODULE=on
export GONOSUMDB="*.green-element-chain.*"
#export GOOS="linux"
export GOPRIVATE="*.green-element-chain.*"
export GOPROXY="https://proxy.golang.org"
export GOSUMDB="off"
installEthereum
installTendermint
installGelchain

