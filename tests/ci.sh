#!/bin/bash

export ROOT_PATH="/tmp/ci_test"
export GOPATH="$ROOT_PATH/go"
export PARENT_DIR="$GOPATH/src/github.com"
export ETHERMINT_PARENT_DIR="$PARENT_DIR/tendermint"
export ETHERMINT_DIR="$ETHERMINT_PARENT_DIR/gelchain"
export TENDERMINT_PARENT_DIR="$PARENT_DIR/tendermint"
export TENDERMINT_DIR="$TENDERMINT_PARENT_DIR/tendermint"
export GETH_PARENT_DIR="$PARENT_DIR/ethereum"
export GETH_DIR="$GETH_PARENT_DIR/go-ethereum"
export DATA_DIR="$ROOT_PATH/.gelchain"

apt update && apt -y install npm

echo 'prepare directory'
rm -rf $ROOT_PATH
docker ps -aq | xargs docker rm -f
mkdir -p $ROOT_PATH/go
mkdir -p $DATA_DIR

echo 'git clone geth'
echo $GETH_PARENT_DIR
mkdir -p $GETH_PARENT_DIR && cd $_
ls $GETH_PARENT_DIR
git clone -b "develop" git@github.com:Green-Element-Chain/go-ethereum.git
go get gopkg.in/urfave/cli.v1

echo 'git clone tendermint'
mkdir -p $TENDERMINT_PARENT_DIR && cd $_
git clone -b "feature/rm_geth_dep" git@github.com:Green-Element-Chain/tendermint.git
cd $TENDERMINT_DIR && make get_tools get_vendor_deps install install_abci

# fix go-amino dep conflict
mv $TENDERMINT_DIR/vendor/github.com/tendermint/go-amino $GOPATH/src/github.com/tendermint/
mv $TENDERMINT_DIR/vendor/github.com/davecgh $GOPATH/src/github.com/

echo 'git clone gelchain'
mkdir -p $ETHERMINT_PARENT_DIR && cd $_
# rm conflict deps
git clone -b "feature/circleci_docker" git@github.com:Green-Element-Chain/ethermint.git
cd $ETHERMINT_DIR
make get_vendor_deps install

cp /root/go/bin/ethermint /usr/bin/ethermint

cd $DATA_DIR && tendermint testnet --v 4 --o ./neweth --populate-persistent-peers --starting-ip-address 172.17.0.101

cd $DATA_DIR && ethermint --datadir mdata init $ETHERMINT_DIR/setup/genesis.json

for((i=0;i<4;i++))
do

  mkdir -p $DATA_DIR/ethermint/chaindata/peer$i && cd $_
  mkdir -p ethermint/chaindata
  cp $ETHERMINT_DIR/setup/genesis.json ./ethermint/chaindata/
  cp -r $DATA_DIR/mdata/* ./
  mkdir -p tendermint
  mkdir -p ethermint
  mv $DATA_DIR/neweth/node$i/config $DATA_DIR/ethermint/chaindata/peer$i/tendermint/

  echo "{\"priv_key\":"$(cat $DATA_DIR/ethermint/chaindata/peer$i/tendermint/config/priv_validator.json | jq ".priv_key")"}"  > $DATA_DIR/ethermint/chaindata/peer$i/tendermint/config/node_key.json

typeset -l address$i
export address$i=$(cat $DATA_DIR/ethermint/chaindata/peer$i/tendermint/config/priv_validator.json | jq ".address" |sed 's/\"//g')

done

for((j=0;j<4;j++))
do
  port1=$((8545 + $j * 10))
  port2=$((46656 + $j * 10))
  port3=$((46657 + $j * 10))
  port4=$((46658 + $j * 10))
  port5=$((19190 + $j * 10))
  docker run -tid --net=bridge --name=peer$j -p $port1:8545 -p $port2:46656 -p $port3:26657 -p $port4:26658 -p $port5:19190 -v $DATA_DIR/ethermint/chaindata/peer$j:/chaindata -v /usr/bin:/bin ubuntu /bin/ethermint --datadir /chaindata --with-tendermint  --rpc --rpccorsdomain=* --rpcvhosts=*  --rpcaddr=0.0.0.0 --ws --wsaddr=0.0.0.0 --rpcapi eth,net,web3,personal,admin,shh --gcmode=full --lightpeers=15 --pex=true --fast_sync=true --priv_validator_file=/chaindata/tendermint/config/priv_validator.json --tendermint_p2paddr=tcp://0.0.0.0:46656 --addr_book_file=addr_book.json --routable_strict=false  --persistent_peers=$address0@172.17.0.2:46656,$address1@172.17.0.3:46656,$address2@172.17.0.4:46656,$address3@172.17.0.5:46656 --logLevel=info

done

echo 'wait 60s ...'
sleep 60

mkdir -p $DATA_DIR/ethermint/ci_test && cp -r $ETHERMINT_DIR/tests/integration/truffle/* $DATA_DIR/ethermint/ci_test/
cd $DATA_DIR/ethermint/ci_test
npm install && npm cache clean --force
npm test
