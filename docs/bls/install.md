# Tendermint with BLS

## DKG+TBLS
BLS signature scheme is a cryptographic algorithm based on bilinear 
mapping.  It can be applied to the design of verifiable random functions, 
with verifiability, randomness, uniqueness and certainty.
For a given threshold signature scheme, the DKG protocol allows several
participants to jointly generate the key of the scheme, namely the group
public key and the public and private key pairs of each person, without
the need for a trusted third party. This project adopts the "Joint Feldman"
DKG algorithm to realize the key generation process of threshold signature
scheme. This part is the core of the whole project, and also the core of
random generation. This project adopts threshold signature scheme (TBLS)
to realize threshold group relay.

## Getting Started

### Deploying BLS environment

The first apps we will work with are written in Go. To install them, you
need to [install Go](https://golang.org/doc/install) and put
`$GOPATH/bin` in your `$PATH`

Deploying BLS environment (centos)
```
sudo yum install -y gmp-devel openssl-devel gcc
mkdir -p $GOPATH/src/github.com/herumi

cd $GOPATH/src/github.com/herumi
git clone https://github.com/DTFN/mcl.git && cd $GOPATH/src/github.com/herumi/mcl
git reset --hard fe95b63cc450bc1eb0459dda916a858b5442a258 && make

cd $GOPATH/src/github.com/herumi
git clone https://github.com/DTFN/bls.git && cd $GOPATH/src/github.com/herumi/bls
git reset --hard e9c72f18ab9bc09923da739151821cc588c0d295 && make
```

Deploying BLS environment (ubuntu)
```
sudo apt-get install -y libgmp-dev libssl-dev openssl gcc
mkdir -p $GOPATH/src/github.com/herumi

cd $GOPATH/src/github.com/herumi
git clone https://github.com/DTFN/mcl.git && cd $GOPATH/src/github.com/herumi/mcl
git reset --hard fe95b63cc450bc1eb0459dda916a858b5442a258 && make

cd $GOPATH/src/github.com/herumi
git clone https://github.com/DTFN/bls.git && cd $GOPATH/src/github.com/herumi/bls
git reset --hard e9c72f18ab9bc09923da739151821cc588c0d295 && make

```

