#!/usr/bin/env bash
installEthereum() {
  if [ -d $GOPATH/src/github.com/ethereum/go-ethereum ]; then
    cd $GOPATH/src/github.com/ethereum/go-ethereum
#    git pull
  else
    mkdir -p $GOPATH/src/github.com/ethereum && cd $GOPATH/src/github.com/ethereum
    pwd
    git clone git@github.com:DTFN/go-ethereum.git
    cd go-ethereum
  fi

  go mod vendor
  cd cmd/geth && go install
  cd ../../../..
}

installTendermint() {
  if [ -d $GOPATH/src/github.com/tendermint/tendermint ]; then
    cd $GOPATH/src/github.com/tendermint/tendermint
#    git pull
  else
    mkdir -p $GOPATH/src/github.com/tendermint && cd $GOPATH/src/github.com/tendermint
    git clone git@github.com:DTFN/tendermint.git
    cd tendermint
  fi

  go mod vendor
  cd cmd/tendermint && go install
  cd ../../../..
}

installGelchain() {
  cd DTFN/dtfn

  go mod vendor
  backupVendors
  export GO111MODULE=off
  cd cmd/gelchain && go install
  cd ../..
  restoreVendors
  export GO111MODULE=on
}

backupVendors() {
  rm vendor_bak -rf
  mkdir vendor_bak
  mkdir -p vendor_bak/ethereum/go-ethereum/vendor/github.com/tendermint
  mv ../../ethereum/go-ethereum/vendor/github.com/tendermint/tendermint vendor_bak/ethereum/go-ethereum/vendor/github.com/tendermint
  mkdir -p vendor_bak/DTFN/gelchain/vendor/github.com/tendermint
  mv vendor/github.com/tendermint/tendermint vendor_bak/DTFN/gelchain/vendor/github.com/tendermint
  mkdir -p vendor_bak/tendermint/tendermint/vendor/github.com/ethereum
  mv ../../tendermint/tendermint/vendor/github.com/ethereum/go-ethereum vendor_bak/tendermint/tendermint/vendor/github.com/ethereum
  mkdir -p vendor_bak/DTFN/gelchain/vendor/github.com/ethereum
  mv vendor/github.com/ethereum/go-ethereum vendor_bak/DTFN/gelchain/vendor/github.com/ethereum
  rm ../../ethereum/go-ethereum/vendor/github.com/karalabe -rf
  rm ../../ethereum/go-ethereum/vendor/gopkg.in -rf
  rm ../../../gopkg.in -rf
  mv vendor/gopkg.in ../../.. -f
}

restoreVendors() {
  mv vendor_bak/ethereum/go-ethereum/vendor/github.com/tendermint/tendermint ../../ethereum/go-ethereum/vendor/github.com/tendermint
  mv vendor_bak/DTFN/gelchain/vendor/github.com/tendermint/tendermint vendor/github.com/tendermint
  mv vendor_bak/tendermint/tendermint/vendor/github.com/ethereum/go-ethereum ../../tendermint/tendermint/vendor/github.com/ethereum
  mv vendor_bak/DTFN/gelchain/vendor/github.com/ethereum/go-ethereum vendor/github.com/ethereum
  cp ../../karalabe ../../ethereum/go-ethereum/vendor/github.com/ -rf
  cp ../../../gopkg.in vendor ../../ethereum/go-ethereum/vendor -rf
  rm vendor_bak -rf
}

export GO111MODULE=on
#export GOOS="linux"

# If you don't want to input the username and password,you could config the git setting
# git config --global url."https://<access-token-here>:x-oauth-basic@github.com/".insteadOf "https://github.com/"
# demo:
# git config --global url."https://eac1464aa42c023c5de8d74ec0a37f3f26568766:x-oauth-basic@github.com/".insteadOf "https://github.com/"
export GIT_TERMINAL_PROMPT=1

unset GOPROXY
go env -w GOPROXY=https://goproxy.cn

go env -w GONOSUMDB=github.com/DTFN/*
go env -w GOPRIVATE=github.com/DTFN/*
go env -w GONOPROXY=github.com/DTFN/*
go env -w GOSUMDB="off"

installEthereum
installTendermint
installGelchain
