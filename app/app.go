package app

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	errors "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/blacklist"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/green-element-chain/gelchain/ethereum"
	"github.com/green-element-chain/gelchain/httpserver"
	emtTypes "github.com/green-element-chain/gelchain/types"
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
	subBalanceStrategy := &BurnStrategy{}
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

// SetLogger sets the logger for the gelchain application
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
	blockchain := app.backend.Ethereum().BlockChain()
	currentBlock := blockchain.CurrentBlock()
	height := currentBlock.Number()
	hash := currentBlock.Header().Hash()

	app.logger.Debug("Info", "height", height) // nolint: errcheck

	// This check determines whether it is the first time gelchain gets started.
	// If it is the first time, then we have to respond with an empty hash, since
	// that is what tendermint expects.
	if height.Cmp(bigZero) == 0 {
		return abciTypes.ResponseInfo{
			Data:             "ABCIEthereum",
			LastBlockHeight:  height.Int64(),
			LastBlockAppHash: []byte{},
		}
	}

	return abciTypes.ResponseInfo{
		Data:             "ABCIEthereum",
		LastBlockHeight:  height.Int64(),
		LastBlockAppHash: hash[:],
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

	app.logger.Debug("InitChain") // nolint: errcheck
	app.SetValidators(req.Validators)
	app.strategy.CurrRoundValData.InitialValidators = req.Validators

	ethState, _ := app.getCurrentState()

	for j := 0; j < len(app.strategy.CurrRoundValData.InitialValidators); j++ {
		app.GetLogger().Info("CurrRoundValData.InitialValidators sige", "blacklist",
			len(app.strategy.CurrRoundValData.InitialValidators))
		pubKey := app.strategy.CurrRoundValData.InitialValidators[j].PubKey
		tmPubKey, _ := types.PB2TM.PubKey(pubKey)
		address := strings.ToLower(tmPubKey.Address().String())
		if app.strategy.CurrRoundValData.AccountMapList.MapList[address] == nil {
			continue
		}
		upsertFlag, _ := app.UpsertPosItemInit(
			app.strategy.CurrRoundValData.AccountMapList.MapList[address].Signer,
			app.strategy.CurrRoundValData.AccountMapList.MapList[address].SignerBalance,
			app.strategy.CurrRoundValData.AccountMapList.MapList[address].Beneficiary,
			app.strategy.CurrRoundValData.InitialValidators[j].PubKey)
		if upsertFlag {
			app.strategy.CurrRoundValData.CurrCandidateValidators = append(app.
				strategy.CurrRoundValData.CurrCandidateValidators, app.
				strategy.CurrRoundValData.InitialValidators[j])

			app.strategy.NextRoundValData.NextRoundCandidateValidators = append(app.
				strategy.NextRoundValData.NextRoundCandidateValidators, app.
				strategy.CurrRoundValData.InitialValidators[j])

			blacklist.Lock(ethState, app.strategy.CurrRoundValData.AccountMapList.MapList[address].Signer)
			app.GetLogger().Info("UpsertPosTable true and Lock initial Account", "blacklist",
				app.strategy.CurrRoundValData.AccountMapList.MapList[address].Signer)
		} else {
			//This is used to remove the validators who dont have enough balance
			//but he is in the accountmap.
			app.strategy.FirstInitial = true
			pubKey := app.strategy.CurrRoundValData.InitialValidators[j].PubKey
			tmPubKey, _ := types.PB2TM.PubKey(pubKey)
			tmAddress := strings.ToLower(tmPubKey.Address().String())
			app.strategy.CurrRoundValData.AccMapInitial.MapList[tmAddress] = app.strategy.CurrRoundValData.AccountMapList.MapList[tmAddress]
			delete(app.strategy.CurrRoundValData.AccountMapList.MapList, tmAddress)
			app.GetLogger().Info("remove not enough balance validators")
		}
	}

	app.logger.Info("deep copy when initChain")
	accByte, _ := json.Marshal(app.strategy.CurrRoundValData.AccountMapList)
	json.Unmarshal(accByte, app.strategy.NextRoundValData.NextAccountMapList)
	posByte, _ := json.Marshal(app.strategy.CurrRoundValData.PosTable)
	json.Unmarshal(posByte, app.strategy.NextRoundValData.NextRoundPosTable)
	app.strategy.NextRoundValData.NextRoundPosTable.ChangedFlagThisBlock = true
	app.PersistenceData()

	return abciTypes.ResponseInitChain{}
}

// CheckTx checks a transaction is valid but does not mutate the state
// #stable - 0.4.0
func (app *EthermintApplication) CheckTx(txBytes []byte) abciTypes.ResponseCheckTx {
	tx, err := decodeTx(txBytes)
	if err != nil {
		// nolint: errcheck
		app.logger.Debug("CheckTx: Received invalid transaction", "tx", tx)
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInternal),
			Log:  err.Error(),
		}
	}
	app.logger.Debug("CheckTx: Received valid transaction", "tx", tx) // nolint: errcheck

	return app.validateTx(tx)
}

// DeliverTx executes a transaction against the latest state
// #stable - 0.4.0
func (app *EthermintApplication) DeliverTx(txBytes []byte) abciTypes.ResponseDeliverTx {
	tx, err := decodeTx(txBytes)
	if err != nil {
		// nolint: errcheck
		app.logger.Debug("DelivexTx: Received invalid transaction", "tx", tx, "err", err)
		return abciTypes.ResponseDeliverTx{
			Code: uint32(errors.CodeInternal),
			Log:  err.Error(),
		}
	}
	app.logger.Debug("DeliverTx: Received valid transaction", "tx", tx) // nolint: errcheck

	res, wrap := app.backend.DeliverTx(tx, app.Receiver())
	if res.IsErr() {
		// nolint: errcheck
		app.logger.Error("DeliverTx: Error delivering tx to ethereum backend", "tx", tx,
			"err", err)
		return res
	}

	db, e := app.getCurrentState()
	if e == nil {
		if wrap.Type == "upsert" {
			b, e := app.UpsertValidatorTx(wrap.Signer, wrap.Height, wrap.Balance, wrap.Beneficiary, wrap.Pubkey, wrap.BlsKeyString)
			if e == nil && b {
				blacklist.Lock(db, wrap.Signer)
			} else {
				eMsg := "error"
				if e != nil {
					eMsg = e.Error()
				}
				wrap.Receipt.Status = ethTypes.ReceiptStatusFailed
				log := &ethTypes.Log{Address: wrap.Signer, Data: []byte(eMsg)}
				wrap.Receipt.Logs = append(wrap.Receipt.Logs, log)
			}
		} else if wrap.Type == "remove" {
			b, e := app.RemoveValidatorTx(wrap.Signer)
			if e == nil && b {
				blacklist.Unlock(db, wrap.Signer, wrap.Height)
			}
		}
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
	header := beginBlock.GetHeader()
	// update the eth header with the tendermint header!br0ken!!
	app.backend.UpdateHeaderWithTimeInfo(&header)

	if !app.strategy.FirstInitial {
		app.logger.Info("delete all current maplist")
		for key, _ := range app.strategy.CurrRoundValData.AccountMapList.MapList {
			delete(app.strategy.CurrRoundValData.AccountMapList.MapList, key)
		}
	}
	app.InitPersistData()
	app.strategy.CurrRoundValData.ProposerAddress = hex.EncodeToString(beginBlock.Header.ProposerAddress)
	app.strategy.CurrRoundValData.Receiver = app.Receiver().String()

	if (header.Height)%200 == 0 {
		app.logger.Info("DeepCopy")
		accByte, _ := json.Marshal(app.strategy.NextRoundValData.NextAccountMapList)

		app.strategy.CurrRoundValData.CurrCandidateValidators = nil
		app.strategy.CurrRoundValData.PosTable = nil
		for key, _ := range app.strategy.CurrRoundValData.AccountMapList.MapList {
			delete(app.strategy.CurrRoundValData.AccountMapList.MapList, key)
		}
		accountMaplist := emtTypes.AccountMapList{}

		json.Unmarshal(accByte, &accountMaplist)
		for key, value := range accountMaplist.MapList {
			app.strategy.CurrRoundValData.AccountMapList.MapList[key] = value
		}

		for i := 0; i < len(app.strategy.NextRoundValData.NextRoundCandidateValidators); i++ {
			app.strategy.CurrRoundValData.CurrCandidateValidators = append(
				app.strategy.CurrRoundValData.CurrCandidateValidators,
				app.strategy.NextRoundValData.NextRoundCandidateValidators[i])
		}

		posByte, _ := json.Marshal(app.strategy.NextRoundValData.NextRoundPosTable)
		postable := emtTypes.PosTable{}
		json.Unmarshal(posByte, &postable)

		app.strategy.CurrRoundValData.PosTable = &postable
		app.strategy.CurrRoundValData.PosTable.PosNodeSortList = emtTypes.NewValSortlist()
	}

	if !app.strategy.FirstInitial {
		// before next bonus ,clear accountMapListTemp
		for key, _ := range app.strategy.CurrRoundValData.AccMapInitial.MapList {
			delete(app.strategy.CurrRoundValData.AccMapInitial.MapList, key)
		}
	} else {
	}

	db, e := app.getCurrentState()
	if e == nil {
		//app.logger.Info("do punish")
		app.punishment.DoPunish(app, db, beginBlock.ByzantineValidators, app.strategy.CurrRoundValData.CurrentValidators)
	}

	return abciTypes.ResponseBeginBlock{}
}

func (app *EthermintApplication) GetAccountMap(tmAddress string) *emtTypes.AccountMap {
	return app.strategy.CurrRoundValData.AccountMapList.MapList[tmAddress]
}

// EndBlock accumulates rewards for the validators and updates them
// #stable - 0.4.0
func (app *EthermintApplication) EndBlock(endBlock abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	if endBlock.GetSeed() != nil {
		app.logger.Debug("EndBlock", "height", endBlock.GetHeight()) // nolint: errcheck
	}

	app.logger.Debug("EndBlock", "height", endBlock.GetSeed()) // nolint: errcheck

	return app.GetUpdatedValidators(endBlock.GetHeight(), endBlock.GetSeed())
}

// Commit commits the block and returns a hash of the current state
// #stable - 0.4.0
func (app *EthermintApplication) Commit() abciTypes.ResponseCommit {

	app.backend.AccumulateRewards(app.strategy)
	app.PersistenceData()

	app.logger.Debug("Commit") // nolint: errcheck
	state, err := app.getCurrentState()
	if err != nil {
		app.logger.Error("Error getting latest state", "err", err) // nolint: errcheck
		return abciTypes.ResponseCommit{}
	}
	app.checkTxState = state.Copy() //commit里会做recheck，需要先重置checkState,通过recheck也正好将checkState恢复到正确的状态
	blockHash, err := app.backend.Commit(app.Receiver())
	if err != nil {
		// nolint: errcheck
		app.logger.Error("Error getting latest ethereum state", "err", err)
		return abciTypes.ResponseCommit{}
	}

	return abciTypes.ResponseCommit{
		Data: blockHash[:],
	}
}

// Query queries the state of the EthermintApplication
// #stable - 0.4.0
func (app *EthermintApplication) Query(query abciTypes.RequestQuery) abciTypes.ResponseQuery {
	app.logger.Debug("Query") // nolint: errcheck
	var in jsonRequest
	if err := json.Unmarshal(query.Data, &in); err != nil {
		return abciTypes.ResponseQuery{Code: uint32(errors.CodeInternal),
			Log: err.Error()}
	}
	var result interface{}
	if err := app.rpcClient.Call(&result, in.Method, in.Params...); err != nil {
		return abciTypes.ResponseQuery{Code: uint32(errors.CodeInternal),
			Log: err.Error()}
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return abciTypes.ResponseQuery{Code: uint32(errors.CodeInternal),
			Log: err.Error()}
	}
	return abciTypes.ResponseQuery{Code: abciTypes.CodeTypeOK, Value: bytes}
}

//-------------------------------------------------------

// validateTx checks the validity of a tx against the blockchain's current state.
// it duplicates the logic in ethereum's tx_pool
func (app *EthermintApplication) validateTx(tx *ethTypes.Transaction) abciTypes.ResponseCheckTx {
	if app.backend.Ethereum().TxPool().ValidateExist(tx.Hash()){
		app.logger.Debug("ValidateTx: Received existed transaction", "txpool", tx.Hash()) // nolint: errcheck
		return abciTypes.ResponseCheckTx{Code: abciTypes.CodeTypeOK}
	}
	// Heuristic limit, reject transactions over 32KB to prevent DOS attacks
	if tx.Size() > maxTransactionSize {
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInternal),
			Log:  core.ErrOversizedData.Error()}
	}

	var signer ethTypes.Signer = ethTypes.FrontierSigner{}
	if tx.Protected() {
		signer = ethTypes.NewEIP155Signer(tx.ChainId())
	}

	// Make sure the transaction is signed properly
	from, err := ethTypes.Sender(signer, tx)
	if err != nil {
		// TODO: Add errors.CodeTypeInvalidSignature ?
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInternal),
			Log:  core.ErrInvalidSender.Error()}
	}

	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInvalidPubKey),
			Log:  core.ErrNegativeValue.Error()}
	}

	currentState := app.checkTxState

	// Make sure the account exist - cant send from non-existing account.
	if !currentState.Exist(from) {
		workState, _ := app.backend.Es().State()
		if !workState.Exist(from) {
			return abciTypes.ResponseCheckTx{
				Code: uint32(errors.CodeUnknownAddress),
				Log:  core.ErrInvalidSender.Error()}
		}
	}

	// Check the transaction doesn't exceed the current block limit gas.
	gasLimit := app.backend.GasLimit()
	if gasLimit < tx.Gas() {
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInternal),
			Log:  core.ErrGasLimitReached.Error()}
	}

	// Check if nonce is not strictly increasing
	nonce := currentState.GetNonce(from)
	if nonce != tx.Nonce() {
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInvalidSequence),
			Log: fmt.Sprintf(
				"Nonce not strictly increasing. Expected %d Got %d",
				nonce, tx.Nonce())}
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	currentBalance := currentState.GetBalance(from)
	if currentBalance.Cmp(tx.Cost()) < 0 {
		return abciTypes.ResponseCheckTx{
			// TODO: Add errors.CodeTypeInsufficientFunds ?
			Code: uint32(errors.CodeInsufficientFunds),
			Log: fmt.Sprintf(
				"Current balance: %s, tx cost: %s",
				currentBalance, tx.Cost())}
	}

	intrGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true) // homestead == true

	if err != nil && tx.Gas() < intrGas {
		return abciTypes.ResponseCheckTx{
			Code: uint32(errors.CodeInsufficientCoins),
			Log:  err.Error()}
	}

	// Update ether balances
	// amount + gasprice * gaslimit
	currentState.SubBalance(from, tx.Cost())
	// tx.To() returns a pointer to a common address. It returns nil
	// if it is a contract creation transaction.
	if to := tx.To(); to != nil {
		currentState.AddBalance(*to, tx.Value())
	}
	currentState.SetNonce(from, tx.Nonce()+1)

	return abciTypes.ResponseCheckTx{Code: abciTypes.CodeTypeOK}
}

func (app *EthermintApplication) GetStrategy() *emtTypes.Strategy {
	return app.strategy
}

func (app *EthermintApplication) UpsertPosItemInit(account common.Address, balance *big.Int, beneficiary common.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	if app.strategy != nil {
		bool, err := app.strategy.CurrRoundValData.PosTable.UpsertPosItem(account, balance, beneficiary, pubkey)
		return bool, err
	}
	return false, nil
}

func (app *EthermintApplication) RemovePosItemInit(account common.Address) (bool, error) {
	if app.strategy != nil {
		bool, err := app.strategy.CurrRoundValData.PosTable.RemovePosItem(account)
		return bool, err
	}
	return false, nil
}
