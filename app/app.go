package app

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/DTFN/dtfn/ethereum"
	"github.com/DTFN/dtfn/httpserver"
	emtTypes "github.com/DTFN/dtfn/types"
	"github.com/DTFN/dtfn/version"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txfilter"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmLog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/types"
	"math/big"
	"strings"
)

// EthermintApplication implements an ABCI application
// #stable - 0.4.0
type EthermintApplication struct {
	// backend handles the ethereum state machine
	// and wrangles other services started by an ethereum node (eg. tx pool)
	backend *ethereum.Backend // backend ethereum struct

	// a closure to return the latest current state from the ethereum blockchain
	getCurrentState func() (*state.StateDB, error)

	checkTxState *state.StateDB

	// an ethereum rpc client we can forward queries to
	rpcClient *rpc.Client

	// strategy for validator compensation
	strategy *emtTypes.Strategy

	// select count
	SelectCount int64

	// select change block number
	SelectBlockNumber int64

	//select Strategy in the test
	SelectStrategy bool

	httpServer *httpserver.BaseServer

	punishment *Punishment

	logger tmLog.Logger
}

// NewEthermintApplication creates a fully initialised instance of EthermintApplication
// #stable - 0.4.0
func NewEthermintApplication(backend *ethereum.Backend,
	client *rpc.Client, strategy *emtTypes.Strategy) (*EthermintApplication, error) {

	state, err := backend.Ethereum().BlockChain().State()
	if err != nil {
		return nil, err
	}

	amountStrategy := &Percent100AmountStrategy{}
	subBalanceStrategy := &TransferStrategy{}
	app := &EthermintApplication{
		backend:         backend,
		rpcClient:       client,
		getCurrentState: backend.Es().State, //backend.Ethereum().BlockChain().State,
		checkTxState:    state.Copy(),
		strategy:        strategy,
		httpServer:      httpserver.NewBaseServer(strategy, backend),
		punishment:      NewPunishment(amountStrategy, subBalanceStrategy),
	}

	if err := app.backend.InitEthState(common.HexToAddress(app.backend.InitReceiver())); err != nil {
		return nil, err
	}

	return app, nil
}

// SetLogger sets the logger for the dtfn application
// #unstable
func (app *EthermintApplication) SetLogger(log tmLog.Logger) {
	app.logger = log
}

func (app *EthermintApplication) GetLogger() tmLog.Logger {
	return app.logger
}

var bigZero = big.NewInt(0)

// maxTransactionSize is 32KB in order to prevent DOS attacks
const maxTransactionSize = 32768

// Info returns information about the last height and app_hash to the tendermint engine
// #stable - 0.4.0

func (app *EthermintApplication) Info(req abciTypes.RequestInfo) abciTypes.ResponseInfo {
	blockChain := app.backend.Ethereum().BlockChain()
	currentBlock := blockChain.CurrentBlock()
	height := currentBlock.Number()
	hash := currentBlock.Header().Hash()
	appVersion := uint64(1)
	if app.strategy.HFExpectedData.IsHarfForkPassed {
		for i := len(version.HeightArray) - 1; i >= 0; i-- {
			if height.Int64() >= version.HeightArray[i] {
				appVersion = uint64(version.VersionArray[i])
				break
			}
		}
	}
	app.logger.Info("Info", "height", height, "appVersion", appVersion) // nolint: errcheck

	minerBonus := big.NewInt(1)
	divisor := big.NewInt(1)
	// for 1% every year increment
	minerBonus.Div(app.strategy.CurrEpochValData.TotalBalance, divisor.Mul(big.NewInt(100), big.NewInt(365*24*60*60/5/2))) //divide 2 is for proposer and voters share half the benefit
	app.strategy.CurrEpochValData.MinorBonus = minerBonus

	// This check determines whether it is the first time dtfn gets started.
	// If it is the first time, then we have to respond with an empty hash, since
	// that is what tendermint expects.
	if height.Cmp(bigZero) == 0 {
		return abciTypes.ResponseInfo{
			Data:             "ABCIEthereum",
			LastBlockHeight:  height.Int64(),
			LastBlockAppHash: []byte{},
			AppVersion:       appVersion,
		}
	}

	return abciTypes.ResponseInfo{
		Data:             "ABCIEthereum",
		LastBlockHeight:  height.Int64(),
		LastBlockAppHash: hash[:],
		AppVersion:       appVersion,
	}
}

// SetOption sets a configuration option
// #stable - 0.4.0
func (app *EthermintApplication) SetOption(req abciTypes.RequestSetOption) abciTypes.ResponseSetOption {
	app.logger.Debug("SetOption", "key", req.GetKey(), "value", req.GetValue()) // nolint: errcheck
	return abciTypes.ResponseSetOption{}
}

// InitChain initializes the validator set
// #stable - 0.4.0
func (app *EthermintApplication) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	app.logger.Info("InitChain", "len(req.Validators)", len(req.Validators)) // nolint: errcheck
	ethState, _ := app.getCurrentState()
	initialValidators := []abciTypes.ValidatorUpdate{}
	app.SetPosTableThreshold()
	if app.strategy.NextEpochValData.PosTable == nil {
		panic("InitChain, app.strategy.NextEpochValData.PosTable==nil. check InitPersistentData")
	}
	for _, validator := range req.Validators {
		pubKey := validator.PubKey
		tmPubKey, _ := types.PB2TM.PubKey(pubKey)
		address := tmPubKey.Address().String()
		if app.strategy.AccMapInitial.MapList[address] == nil {
			app.logger.Error(fmt.Sprintf("initChain address %v not found in initialAccountMap, ignore. Please check configuration!", address))
			continue
		}
		signer := app.strategy.AccMapInitial.MapList[address].Signer
		signerBalance := ethState.GetBalance(signer)
		err := app.InsertPosItemInit(
			signer,
			signerBalance,
			app.strategy.AccMapInitial.MapList[address].Beneficiary,
			validator.PubKey,
			app.strategy.AccMapInitial.MapList[address].BlsKeyString)
		if err == nil {
			initialValidators = append(app.strategy.InitialValidators, validator)
			app.strategy.CurrentHeightValData.Validators[address] = emtTypes.Validator{
				validator,
				signer,
			} //In height 1, we will start delete validators

			app.GetLogger().Info("UpsertPosTable true and Lock initial Account", "blacklist", signer)
		} else {
			//This is used to remove the validators who dont have enough balance
			//but he is in the accountmap.
			delete(app.strategy.AccMapInitial.MapList, address)
			app.GetLogger().Error(fmt.Sprintf("remove not enough balance validator %v.  err %v", app.strategy.AccMapInitial.MapList[address], err))
		}
	}
	initialValidatorsLen := len(initialValidators)
	app.strategy.SetInitialValidators(initialValidators)
	app.logger.Info("InitialValidators", "len(app.strategy.InitialValidators)", initialValidatorsLen,
		"validators", app.strategy.InitialValidators)
	if initialValidatorsLen != 0 {
		app.strategy.NextEpochValData.PosTable.InitStruct()
		app.strategy.CurrEpochValData.PosTable = app.strategy.NextEpochValData.PosTable.Copy()
		txfilter.CurrentPosTable = app.strategy.CurrEpochValData.PosTable
		txfilter.EthAuthTableCopy = txfilter.EthAuthTable.Copy()
		app.strategy.CurrEpochValData.PosTable.ExportSortedSigners()
	} else {
		panic("no qualified initial validators, please check config")
	}

	app.SetPersistenceData()

	return abciTypes.ResponseInitChain{}
}

// CheckTx checks a transaction is valid but does not mutate the state
// #stable - 0.4.0
func (app *EthermintApplication) CheckTx(req abciTypes.RequestCheckTx) abciTypes.ResponseCheckTx {
	var tx *ethTypes.Transaction
	if req.Type == abciTypes.CheckTxType_Local {
		tx = app.backend.CurrentTxInfo().Tx
	} else {
		txBytes := req.Tx
		var err error
		tx, err = decodeTx(txBytes)
		if err != nil {
			// nolint: errcheck
			app.logger.Debug("CheckTx: Received invalid transaction", "tx", tx)
			return abciTypes.ResponseCheckTx{
				Code: uint32(emtTypes.CodeInternal),
				Log:  err.Error(),
			}
		}
	}

	app.logger.Debug("CheckTx: Received valid transaction", "tx", tx) // nolint: errcheck

	return app.validateTx(tx, req.Type)
}

// DeliverTx executes a transaction against the latest state
// #stable - 0.4.0
func (app *EthermintApplication) DeliverTx(req abciTypes.RequestDeliverTx) abciTypes.ResponseDeliverTx {
	tx, err := decodeTx(req.Tx)
	if err != nil {
		// nolint: errcheck
		app.logger.Debug("DelivexTx: Received invalid transaction", "tx", tx, "err", err)
		return abciTypes.ResponseDeliverTx{
			Code: uint32(emtTypes.CodeInternal),
			Log:  err.Error(),
		}
	}

	txHash := tx.Hash()
	txInfo, ok := app.backend.FetchCachedTxInfo(txHash)
	if !ok {
		var signer ethTypes.Signer = ethTypes.HomesteadSigner{}
		if tx.Protected() {
			signer = app.strategy.Signer()
		}
		if tx.To() != nil {
			if txfilter.IsRelayTxFromRelayer(*tx.To()) {
				txInfo = ethTypes.TxInfo{Tx: tx}
				txInfo.SubTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx decode failed. %v",
							core.ErrInvalidSender.Error())}
				}
				err = ethTypes.CheckRelayerTx(tx, txInfo.SubTx)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Relayer tx not match with main tx, please check, %v", err)}
				}
				txForVerify, err := txInfo.SubTx.WithVRS(tx.RawSignatureValues())
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx WithVRS failed. %v",
							core.ErrInvalidSender.Error())}
				}
				txInfo.From, err = ethTypes.Sender(signer, txForVerify)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
				tx.SetFrom(signer, txInfo.From)
				txInfo.RelayFrom, err = ethTypes.DeriveRelayer(txInfo.From, txInfo.SubTx)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer signature verified failed. %v",
							core.ErrInvalidSender.Error())}
				}
			} else {
				// Make sure the transaction is signed properly
				from, err := ethTypes.Sender(signer, tx)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
				txInfo = ethTypes.TxInfo{Tx: tx, From: from}

				if txfilter.IsRelayTxFromClient(*tx.To()) {
					txInfo.SubTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
					if err != nil {
						return abciTypes.ResponseDeliverTx{
							Code: uint32(emtTypes.CodeInternal),
							Log: fmt.Sprintf("relayer sub tx decode failed. %v",
								core.ErrInvalidSender.Error())}
					}
					err = ethTypes.CheckRelayerTx(tx, txInfo.SubTx)
					if err != nil {
						return abciTypes.ResponseDeliverTx{
							Code: uint32(emtTypes.CodeInvalidSequence),
							Log: fmt.Sprintf(
								"Relayer tx not match with main tx, please check, %v", err)}
					}
					txInfo.RelayFrom, err = ethTypes.DeriveRelayer(from, txInfo.SubTx)
					if err != nil {
						return abciTypes.ResponseDeliverTx{
							Code: uint32(emtTypes.CodeInternal),
							Log: fmt.Sprintf("relayer signature verified failed. %v",
								core.ErrInvalidSender.Error())}
					}
				}
			}
		} else {
			// Make sure the transaction is signed properly
			from, err := ethTypes.Sender(signer, tx)
			if err != nil {
				return abciTypes.ResponseDeliverTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrInvalidSender.Error()}
			}
			txInfo = ethTypes.TxInfo{Tx: tx, From: from}
		}
	} else {
		app.backend.DeleteCachedTxInfo(txHash)
	}

	res := app.backend.DeliverTx(tx, app.strategy.HFExpectedData.BlockVersion, txInfo)
	if res.IsErr() {
		// nolint: errcheck
		app.logger.Error("DeliverTx: Error delivering tx to ethereum backend", "tx", tx,
			"err", err)
		return res
	}
	//app.CollectTx(tx)
	return abciTypes.ResponseDeliverTx{
		Code: abciTypes.CodeTypeOK,
	}
}

// BeginBlock starts a new Ethereum block
// #stable - 0.4.0
func (app *EthermintApplication) BeginBlock(beginBlock abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {
	app.logger.Debug("BeginBlock") // nolint: errcheck
	app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock = false
	header := beginBlock.GetHeader()
	// update the eth header with the tendermint header!breaking!!
	app.backend.UpdateHeaderWithTimeInfo(&header)
	app.strategy.HFExpectedData.Height = beginBlock.GetHeader().Height
	app.strategy.HFExpectedData.BlockVersion = beginBlock.GetHeader().Version.App
	app.strategy.CurrentHeightValData.Height = beginBlock.GetHeader().Height
	//when we reach the upgrade height,we change the blockversion

	if app.strategy.HFExpectedData.IsHarfForkPassed {
		for i := len(version.HeightArray) - 1; i >= 0; i-- {
			if app.strategy.HFExpectedData.Height >= version.HeightArray[i] {
				app.strategy.HFExpectedData.BlockVersion = uint64(version.VersionArray[i])
				break
			}
		}
	}
	app.logger.Info("block version", "appVersion", app.strategy.HFExpectedData.BlockVersion)
	txfilter.AppVersion = app.strategy.HFExpectedData.BlockVersion
	//if app.strategy.HFExpectedData.IsHarfForkPassed && app.strategy.HFExpectedData.Height == version.NextHardForkHeight {
	//	app.strategy.HFExpectedData.BlockVersion = version.NextHardForkVersion
	//}

	app.strategy.CurrentHeightValData.ProposerAddress = strings.ToUpper(hex.EncodeToString(beginBlock.Header.ProposerAddress))
	coinbase := app.Receiver()
	app.backend.Es().UpdateHeaderCoinbase(coinbase)
	app.strategy.CurrentHeightValData.LastVoteInfo = beginBlock.LastCommitInfo.Votes

	db, e := app.getCurrentState()
	if e == nil {
		//app.logger.Info("do punish")
		app.punishment.DoPunish(db, app.strategy, beginBlock.ByzantineValidators, coinbase, beginBlock.Header.Height)
	}
	storedcfg := app.backend.Ethereum().BlockChain().Config()
	fmt.Printf("-------currentheight chainconfig %v rules %v \n", storedcfg, storedcfg.Rules(big.NewInt(beginBlock.Header.Height)))
	return abciTypes.ResponseBeginBlock{}
}

// EndBlock accumulates rewards for the validators and updates them
// #stable - 0.4.0
func (app *EthermintApplication) EndBlock(endBlock abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	app.logger.Info(fmt.Sprintf("EndBlock height %v seed %X ", endBlock.GetHeight(), endBlock.GetSeed())) // nolint: errcheck

	height := endBlock.Height
	if height%txfilter.EpochBlocks == 0 {
		//DeepCopy
		app.strategy.CurrEpochValData.PosTable = app.strategy.NextEpochValData.PosTable.Copy()
		app.strategy.CurrEpochValData.PosTable.ExportSortedSigners()
		txfilter.CurrentPosTable = app.strategy.CurrEpochValData.PosTable
		txfilter.EthAuthTableCopy = txfilter.EthAuthTable.Copy()
		count := app.strategy.NextEpochValData.PosTable.TryRemoveUnbondPosItems(app.strategy.CurrentHeightValData.Height, app.strategy.CurrEpochValData.PosTable.SortedUnbondSigners)
		app.GetLogger().Info(fmt.Sprintf("total remove %d Validators.", count))

		if height == version.HeightArray[3] { //force update genesis config to Constantinople
			db := app.backend.Ethereum().ChainDb()
			stored := rawdb.ReadCanonicalHash(db, 0)
			if (stored == common.Hash{}) {
				app.logger.Error("No genesis block! No need to reset config!")
			} else {
				storedcfg := app.backend.Ethereum().BlockChain().Config()
				upgradeConfig := params.AllEthashProtocolChanges
				upgradeConfig.ChainID = storedcfg.ChainID
				upgradeConfig.Ethash = storedcfg.Ethash //we do not use ethash
				upgradeConfig.Clique = storedcfg.Clique //we do not use clique either
				fmt.Printf("storedcfg %v \n updated to \n upgradeConfig %v \n", storedcfg, upgradeConfig)
				app.backend.Ethereum().BlockChain().SetConfig(upgradeConfig)
				rawdb.WriteChainConfig(db, stored, upgradeConfig)
			}
		}
	}

	if app.strategy.HFExpectedData.BlockVersion >= 6 {
		//we should clear WaitForDeleteMap every block.
		if app.strategy.FrozeTable.ThisBlockChangedFlag {
			for key, _ := range app.strategy.FrozeTable.WaitForDeleteMap {
				delete(app.strategy.FrozeTable.FrozeItemMap, key)
				delete(app.strategy.FrozeTable.WaitForDeleteMap, key)
			}
			txfilter.EthFrozeTableCopy = txfilter.EthFrozeTable.Copy()
			app.strategy.FrozeTable.ThisBlockChangedFlag = false
		}
	}
	return app.GetUpdatedValidators(endBlock.GetHeight(), endBlock.GetSeed())
}

// Commit commits the block and returns a hash of the current state
// #stable - 0.4.0
func (app *EthermintApplication) Commit() abciTypes.ResponseCommit {

	app.backend.AccumulateRewards(app.strategy)
	app.SetPersistenceData()

	state, err := app.getCurrentState()
	if err != nil {
		app.logger.Error("Error getting latest state", "err", err) // nolint: errcheck
		return abciTypes.ResponseCommit{}
	}
	/*app.logger.Debug(fmt.Sprintf("Commit trie.root=%X",state.Trie().Hash()))
	app.logger.Debug(fmt.Sprintf("current=%v next=%v",app.strategy.CurrHeightValData,
		app.strategy.NextEpochValData))
	state.Finalise(true)
	app.logger.Debug(fmt.Sprintf("After finalise Commit trie.root=%X",state.Trie().Hash()))*/
	app.checkTxState = state.Copy() //commit里会做recheck，需要先重置checkState,通过recheck也正好将checkState恢复到正确的状态
	blockHash, err := app.backend.Commit()
	if err != nil {
		// nolint: errcheck
		app.logger.Error("Error getting latest ethereum state", "err", err)
		return abciTypes.ResponseCommit{}
	}

	if app.backend.Ethereum().TxPool().IsFlowControlOpen() {
		memPool := app.backend.MemPool()
		if memPool != nil { //when in replay, memPool has not been set, it is nil
			app.backend.Ethereum().TxPool().SetFlowLimit(memPool.SizeSnapshot())
		}
	}

	return abciTypes.ResponseCommit{
		Data: blockHash[:],
	}
}

// Query queries the state of the EthermintApplication
// #stable - 0.4.0
func (app *EthermintApplication) Query(query abciTypes.RequestQuery) abciTypes.ResponseQuery {
	var in jsonRequest
	if err := json.Unmarshal(query.Data, &in); err != nil {
		return abciTypes.ResponseQuery{Code: uint32(emtTypes.CodeInternal),
			Log: err.Error()}
	}
	var result interface{}
	if index := strings.Index(query.Path, "PosTable"); index >= 0 {
		if query.Path == "PosTable/GetCurrentPosTable" {
			result = app.strategy.CurrEpochValData.PosTable.PosItemMap
		} else if query.Path == "PosTable/GetNextPosTable" {
			result = app.strategy.NextEpochValData.PosTable.PosItemMap
		} else { //default
			result = app.strategy.NextEpochValData.PosTable.PosItemMap
		}
	} else if index := strings.Index(query.Path, "AuthTable"); index >= 0 {
		if query.Path == "AuthTable/GetAuthTable" {
			result = txfilter.EthAuthTable.AuthItemMap
		} else { //default
			result = txfilter.EthAuthTable.AuthItemMap
		}
	} else if index := strings.Index(query.Path, "/p2p/whitelist"); index >= 0 {
		authTableMap := make(map[string]int64)
		for key, ai := range txfilter.EthAuthTable.AuthItemMap {
			if tmAddr, exist := txfilter.EthAuthTable.ExtendAuthTable.SignerToTmAddressMap[key]; exist {
				authTableMap[tmAddr] = ai.PermitHeight
			} else {
				fmt.Printf("Error! account %X not found tmAddr! \n", key)
			}
		}
		for _, pi := range app.strategy.NextEpochValData.PosTable.UnbondPosItemMap {
			authTableMap[pi.TmAddress] = pi.Height
		}
		for _, pi := range app.strategy.NextEpochValData.PosTable.PosItemMap {
			authTableMap[pi.TmAddress] = pi.Height
		}
		result = authTableMap
	} else if index := strings.Index(query.Path, "FrozeTable/GetFrozeTable"); index >= 0 {
		result = txfilter.EthFrozeTableCopy.FrozeItemMap
	} else {
		if err := app.rpcClient.Call(&result, in.Method, in.Params...); err != nil {
			return abciTypes.ResponseQuery{Code: uint32(emtTypes.CodeInternal),
				Log: err.Error()}
		}
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return abciTypes.ResponseQuery{Code: uint32(emtTypes.CodeInternal),
			Log: err.Error()}
	}
	return abciTypes.ResponseQuery{Code: abciTypes.CodeTypeOK, Value: bytes, Info: string(bytes)}
}

//-------------------------------------------------------

// validateTx checks the validity of a tx against the blockchain's current state.
// it duplicates the logic in ethereum's tx_pool
func (app *EthermintApplication) validateTx(tx *ethTypes.Transaction, checkType abciTypes.CheckTxType) abciTypes.ResponseCheckTx {
	// Heuristic limit, reject transactions over 32KB to prevent DOS attacks
	if tx.Size() > maxTransactionSize {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInternal),
			Log:  core.ErrOversizedData.Error()}
	}

	var signer ethTypes.Signer = ethTypes.HomesteadSigner{}
	if tx.Protected() {
		signer = app.strategy.Signer()
	}

	var from, relayer common.Address
	var txInfo ethTypes.TxInfo
	var cached bool
	success := false
	isRelayTx := false
	txHash := tx.Hash()
	if checkType == abciTypes.CheckTxType_Local {
		txInfo = app.backend.CurrentTxInfo()
		from = txInfo.From
		if txInfo.SubTx != nil {
			relayer = txInfo.RelayFrom
			isRelayTx = true
		}
	} else if checkType == abciTypes.CheckTxType_Recheck {
		txInfo, cached = app.backend.FetchCachedTxInfo(txHash)
		if !cached {
			panic(fmt.Sprintf("The from address of tx should stay in cached"))
		} else {
			defer func() {
				if !success {
					app.backend.DeleteCachedTxInfo(txHash)
				}
			}()
		}
		from = txInfo.From
		isRelayTx = txInfo.SubTx != nil
		if isRelayTx {
			relayer = txInfo.RelayFrom
		}
	} else {
		var subTx *ethTypes.Transaction
		if tx.To() != nil {
			if txfilter.IsRelayTxFromClient(*tx.To()) {
				var err error
				// Make sure the transaction is signed properly
				from, err = ethTypes.Sender(signer, tx)
				if err != nil {
					// TODO: Add emtTypes.CodeTypeInvalidSignature ?
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
				subTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx decode failed. %v",
							core.ErrInvalidSender.Error())}
				}
				relayer, err = ethTypes.DeriveRelayer(from, subTx)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer signature verified failed. %v",
							core.ErrInvalidSender.Error())}
				}
				err = ethTypes.CheckRelayerTx(tx, subTx)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Relayer tx not match with main tx, please check, %v", err)}
				}
				isRelayTx = true
			} else if txfilter.IsRelayTxFromRelayer(*tx.To()) {
				var err error
				subTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx decode failed. %v",
							core.ErrInvalidSender.Error())}
				}
				err = ethTypes.CheckRelayerTx(tx, subTx)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Relayer tx not match with main tx, please check, %v", err)}
				}
				txForVerify, err := subTx.WithVRS(tx.RawSignatureValues())
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx WithVRS failed. %v",
							core.ErrInvalidSender.Error())}
				}
				from, err = ethTypes.Sender(signer, txForVerify)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
				tx.SetFrom(signer, from)
				relayer, err = ethTypes.DeriveRelayer(from, subTx)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer signature verified failed. %v",
							core.ErrInvalidSender.Error())}
				}
				isRelayTx = true
			} else {
				var err error
				// Make sure the transaction is signed properly
				from, err = ethTypes.Sender(signer, tx)
				if err != nil {
					// TODO: Add emtTypes.CodeTypeInvalidSignature ?
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
			}
		} else {
			var err error
			// Make sure the transaction is signed properly
			from, err = ethTypes.Sender(signer, tx)
			if err != nil {
				// TODO: Add emtTypes.CodeTypeInvalidSignature ?
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrInvalidSender.Error()}
			}
		}
		txInfo = ethTypes.TxInfo{Tx: tx, From: from, SubTx: subTx, RelayFrom: relayer}
	}

	//check the legalcy of frozetx
	if tx.To() != nil {
		if txfilter.IsFrozeTx(*tx.To()) {
			err := txfilter.ValidateFrozeTx(from, tx.Data())
			if err != nil {
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrInvalidFrozeData.Error()}
			}
		}
	}

	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInvalidCoins),
			Log:  core.ErrNegativeValue.Error()}
	}

	currentState := app.checkTxState

	// Make sure the account exist - cant send from non-existing account.
	if checkType != abciTypes.CheckTxType_Local && !currentState.Exist(from) {
		app.logger.Info(fmt.Sprintf("receive a remote tx with not existed from %X", from))
		/*return abciTypes.ResponseCheckTx{
		Code: uint32(emtTypes.CodeUnknownAddress),
		Log:  core.ErrInvalidSender.Error()}*/
	}

	// Check the transaction doesn't exceed the current block limit gas.
	gasLimit := app.backend.GasLimit()
	if gasLimit < tx.Gas() {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInternal),
			Log:  core.ErrGasLimitReached.Error()}
	}

	// Check if nonce is not strictly increasing
	nonce := currentState.GetNonce(from)
	if nonce != tx.Nonce() {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInvalidSequence),
			Log: fmt.Sprintf(
				"Nonce for %X not strictly increasing. Expected %d Got %d .",
				from, nonce, tx.Nonce())}
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	var currentBalance *big.Int
	if isRelayTx {
		currentBalance = currentState.GetBalance(relayer)
		fmt.Printf("checkTx, using relayer %X balance \n", relayer)
	} else {
		currentBalance = currentState.GetBalance(from)
	}

	if currentBalance.Cmp(tx.Cost()) < 0 {
		return abciTypes.ResponseCheckTx{
			// TODO: Add emtTypes.CodeTypeInsufficientFunds ?
			Code: uint32(emtTypes.CodeInsufficientFunds),
			Log: fmt.Sprintf(
				"Current balance: %s, tx cost: %s",
				currentBalance, tx.Cost())}
	}

	if app.strategy.HFExpectedData.BlockVersion >= 6 {
		if isRelayTx {
			if txfilter.IsFrozed(relayer) {
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrFrozedAddress.Error()}
			}
			if txfilter.IsFrozeBlocked(from, tx.To()) != nil {
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrFrozedAddress.Error()}
			}
		} else {
			if txfilter.IsFrozeBlocked(from, tx.To()) != nil {
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrFrozedAddress.Error()}
			}
		}
	}

	intrGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true) // homestead == true

	if err != nil && tx.Gas() < intrGas {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInsufficientCoins),
			Log:  err.Error()}
	}
	height := app.backend.Es().WorkState().Height()
	err = txfilter.IsBetBlocked(from, tx.To(), currentBalance, tx.Data(), height, false)
	if err != nil {
		return abciTypes.ResponseCheckTx{
			// TODO: Add emtTypes.CodeTypeTxIsBlocked ?
			Code: uint32(emtTypes.CodeInvalidAddress),
			Log: fmt.Sprintf(
				"Tx is blocked: %v",
				err)}
	}

	if tx.To() != nil {
		if txfilter.IsAuthTx(*tx.To()) {
			err := txfilter.IsAuthBlocked(from, tx.Data(), height, false)
			if err != nil {
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInvalidSequence),
					Log: fmt.Sprintf(
						"Auth tx failed, %v", err)}
			}
			currentState.SubBalance(from, tx.Cost())
		} else {
			if isRelayTx {
				currentState.SubBalance(relayer, tx.Cost())
			} else if txfilter.IsMintTx(*tx.To()) {
				err := txfilter.IsMintBlocked(from)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Mint tx failed, %v", err)}
				}
				currentState.SubBalance(from, tx.Cost())
			}
		}
	} else {
		currentState.SubBalance(from, tx.Cost())
	}
	// Update ether balances
	// amount + gasprice * gaslimit

	// tx.To() returns a pointer to a common address. It returns nil
	// if it is a contract creation transaction.
	if to := tx.To(); to != nil && !txfilter.IsMintTx(*tx.To()) {
		currentState.AddBalance(*to, tx.Value())
	}
	currentState.SetNonce(from, tx.Nonce()+1)

	if !cached {
		app.backend.InsertCachedTxInfo(txHash, txInfo)
	}
	success = true
	return abciTypes.ResponseCheckTx{Code: abciTypes.CodeTypeOK, GasWanted: int64(intrGas)}
}

func (app *EthermintApplication) GetStrategy() *emtTypes.Strategy {
	return app.strategy
}

func (app *EthermintApplication) InsertPosItemInit(account common.Address, balance *big.Int, beneficiary common.Address,
	pubKey abciTypes.PubKey, blsKeyString string) error {
	if app.strategy != nil {
		tmpSlot := big.NewInt(0)
		if app.strategy.HFExpectedData.BlockVersion >= 4 {
			tmpSlot = big.NewInt(10)
		} else {
			tmpSlot.Div(balance, app.strategy.NextEpochValData.PosTable.Threshold)
		}
		tmPubKey, _ := types.PB2TM.PubKey(pubKey)
		tmAddress := tmPubKey.Address().String()
		return app.strategy.NextEpochValData.PosTable.InsertPosItem(account, txfilter.NewPosItem(1, tmpSlot.Int64(), pubKey, tmAddress, blsKeyString, beneficiary))
	}
	return nil
}
