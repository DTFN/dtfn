package app

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	errors "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txfilter"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/green-element-chain/gelchain/ethereum"
	"github.com/green-element-chain/gelchain/httpserver"
	emtTypes "github.com/green-element-chain/gelchain/types"
	"github.com/green-element-chain/gelchain/version"
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
	blockChain := app.backend.Ethereum().BlockChain()
	currentBlock := blockChain.CurrentBlock()
	height := currentBlock.Number()
	hash := currentBlock.Header().Hash()
	appVersion := uint64(1)

	app.logger.Debug("Info", "height", height) // nolint: errcheck

	minerBonus := big.NewInt(1)
	divisor := big.NewInt(1)
	// for 1% every year increment
	minerBonus.Div(app.strategy.CurrEpochValData.TotalBalance, divisor.Mul(big.NewInt(100), big.NewInt(365*24*60*60/5)))
	app.strategy.CurrEpochValData.MinorBonus = minerBonus

	// This check determines whether it is the first time gelchain gets started.
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
	app.SetValidators(req.Validators)                                        //old code
	ethState, _ := app.getCurrentState()
	app.strategy.InitialValidators = []abciTypes.ValidatorUpdate{}

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
		err := app.UpsertPosItemInit(
			signer,
			signerBalance,
			app.strategy.AccMapInitial.MapList[address].Beneficiary,
			validator.PubKey,
			app.strategy.AccMapInitial.MapList[address].BlsKeyString)
		if err == nil {
			app.strategy.InitialValidators = append(app.strategy.InitialValidators, validator)
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
	app.logger.Info("InitialValidators", "len(app.strategy.InitialValidators)", len(app.strategy.InitialValidators),
		"validators", app.strategy.InitialValidators)
	app.SetPersistenceData()

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

	res := app.backend.DeliverTx(tx, app.Receiver())
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
	app.logger.Info("block version", "appVersion", app.strategy.HFExpectedData.BlockVersion)

	app.strategy.CurrentHeightValData.Height = beginBlock.GetHeader().Height
	//when we reach the upgrade height,we change the blockversion
	if app.strategy.HFExpectedData.IsHarfForkPassed && app.strategy.HFExpectedData.Height == version.NextHardForkHeight {
		app.strategy.HFExpectedData.BlockVersion = version.NextHardForkVersion
	}

	app.strategy.CurrentHeightValData.ProposerAddress = strings.ToUpper(hex.EncodeToString(beginBlock.Header.ProposerAddress))
	app.backend.Es().UpdateHeaderCoinbase(app.Receiver())
	app.strategy.CurrentHeightValData.LastVoteInfo = beginBlock.LastCommitInfo.Votes

	db, e := app.getCurrentState()
	if e == nil {
		//app.logger.Info("do punish")
		app.punishment.DoPunish(db, app.strategy, beginBlock.ByzantineValidators, beginBlock.Header.ProposerAddress, beginBlock.Header.Height)
	}

	return abciTypes.ResponseBeginBlock{}
}

// EndBlock accumulates rewards for the validators and updates them
// #stable - 0.4.0
func (app *EthermintApplication) EndBlock(endBlock abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	if endBlock.GetSeed() != nil {
		app.logger.Debug("EndBlock", "height", endBlock.GetHeight()) // nolint: errcheck
	}

	app.logger.Debug("EndBlock", "height", endBlock.GetSeed()) // nolint: errcheck

	height := endBlock.Height
	if height%txfilter.EpochBlocks == 0 && height != 0 { //height==0 is when initChain calls this func
		app.TryRemoveValidatorTxs()
		//DeepCopy
		app.strategy.CurrEpochValData.PosTable = app.strategy.NextEpochValData.PosTable.Copy()
		app.strategy.CurrEpochValData.PosTable.ExportSortedSigners()
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
	pubKey abciTypes.PubKey, blsKeyString string) error {
	if app.strategy != nil {
		tmpSlot := big.NewInt(0)
		tmpSlot.Div(balance, app.strategy.CurrEpochValData.PosTable.Threshold)
		tmPubKey, _ := types.PB2TM.PubKey(pubKey)
		tmAddress := tmPubKey.Address().String()
		err := app.strategy.CurrEpochValData.PosTable.UpsertPosItem(account, txfilter.NewPosItem(1, tmpSlot.Int64(), pubKey, tmAddress, blsKeyString, beneficiary))
		if err != nil {
			return err
		}
		err = app.strategy.NextEpochValData.PosTable.UpsertPosItem(account, txfilter.NewPosItem(1, tmpSlot.Int64(), pubKey, tmAddress, blsKeyString, beneficiary))
		return err
	}
	return nil
}
