<div align="center">
  <p>:zap::zap::zap: DTFN : Leading self-dependent innovation  blockchain infrastructure :zap::zap::zap:</p>
  <p>Powered by <a href="https://github.com/ethereum/go-ethereum">Go-Ethereum</a> / <a href="https://github.com/tendermint/tendermint">Tendermint</a> / <a href="https://github.com/cosmos/ethermint">Ethermint</a></p>
</div>


### Building the source
For prerequisites and detailed build instructions please read the Installation Instructions on the wiki.

Building geth requires both a Go (version 1.9 or later) and a C compiler. You can install them using your favourite package manager. Once the dependencies are installed, run
```
cd scripts
source develop_env_ubuntu.sh
source get_dep.sh
source buildAll.sh
```

### Running with docker

One of the quickest ways to get dtfn up and running on your machine is by using Docker:

```
docker run -tid --name=peer0 -p 8545:8545 -p 46656:46656 -p 46657:26657 -p 46658:26658  -v /root/neweth/peer0:/chaindata  webbshi/dtfn:v1.0.0-alpha dtfn --datadir /chaindata --with-tendermint  --rpc --rpccorsdomain=* --rpcvhosts=*  --rpcaddr=0.0.0.0 --ws --wsaddr=0.0.0.0 --rpcapi eth,net,web3,personal,admin,shh --gcmode=full --lightpeers=15 --pex=true --fast_sync=true --priv_validator_file=config/priv_validator.json --initial_eth_account=config/initial_eth_account.json --trie_time_limit=1 --tendermint_p2paddr=tcp://0.0.0.0:46656 --tm_cons_emptyblock=true --tm_cons_eb_inteval=30 --need_proof_block=false --addr_book_file=addr_book.json  --routable_strict=false --logLevel=info
```

### Contribution

Thank you for considering to help out with the source code! We welcome contributions from
anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to dtfn, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base.

### License


The DTFN is licensed under the [GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html).
www.nenglian.com
