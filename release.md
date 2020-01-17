##PPC
#### 1.fix develop_build
#### 2.add upgrade strategy
#### 3.change bonus strategy
- 3.1 cancle block rewards
- 3.2 change bonus strategy to big guy
- 3.3 support mint strategy for big guy
#### 4. ignore tx-value
#### 5. save receipt
- 5.1 use gasused to record all gas spent(miner bonus)
- 5.2 persist postable of specify height to get the relation of ethaccount and tmaccount

#### 6. select strategy
- 6.1 support multi-bet-tx
- 6.2 change select strategy and support upgrade
- 6.3 add ppc_filter, ignore solt judge
- 6.4 forbid origin bet tx, only accept multi-bet-tx
- 6.5 ignore txpool-isBlocked in go-ethereum for upgrade

#### 7. relay
- 7.1 support contract relay, usage is in contract_java_service
- 7.2 support relay-tx not by contract

#### 8 postalbe
- 8.1 reset slots of positem and totlaSlots in nextEpochData and let `currentSlot = int(10)`

#### 9 Bet
- 9.1 create a new structure named PPCTxData
- 9.2 create a new structure named PPCSignature include PPCTxDataStr
- 9.3 implement the PPChain

#### 10 merge
- 10.1 merge develop and 0.31 branch

#### 12 validateTx and returnErr
- 12.1 add validatePPCTx interface in txpool.go
- 12.2 add validatePPCTX interface in gelchain checkTx
- 12.3 add implement in interface,verify relay-contract, bigguy-approved,bet-tx three types.
- 12.4 replace txdata by txdatahash in ppccatable

#### 13 config upgradeHeight and bigguyAddress and others
- 13.1 config upgradeHeight and bigguyAddress in gelchain
- 13.2 remove specifyHeightPosTable data that we don't need to persist
- 13.3 change upgradeHeight from *int64 to int64 in go-ethereum
- 13.4 fix nonce bug in kickout tx of bigguy
- 13.5 bigguy account may sent bet-tx and as relayer and deploy contract
- 13.6 fix contract event bug when we don't want to relay
- 13.7 we didn't need reset the slots of postable because it it right now.
- 13.8 add GetPPCCATABLE interface and fix persist ppcTableItem.User bug
- 13.9 remove fmt.Println
- 13.10 handle error data in new-tx
- 13.11 add PPCCachedTx for performance
- 13.12 add PPCIllegalRelayForm check
- 13.13 remove key-value in PPCTXFilterCached
- 13.14 for restart error when some-tx was in tx-pool
- 13.15 remove print-log
- 13.16 rename pendingTxPreEvents to currentTxPreEvent
- 13.17 change the cacheTx strategy for not-local tx
- 13.18 add setSubFromNonce in validateTx
- 13.19 reduce the times of verify signature
- 13.20 remove check Nonce in deliverTx
#### testcase


