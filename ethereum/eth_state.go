package ethereum

import (
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"

	abciTypes "github.com/tendermint/tendermint/abci/types"

	"encoding/hex"
	"github.com/ethereum/go-ethereum/log"
	emtTypes "github.com/green-element-chain/gelchain/types"
	"time"
)

const errorCode = 1

//----------------------------------------------------------------------
// EthState manages concurrent access to the intermediate workState object
// The ethereum tx pool fires TxPreEvent in a go-routine,
// and the miner subscribes to this in another go-routine and processes the tx onto
// an intermediate state. We used to use `unsafe` to overwrite the miner, but this
// didn't work because it didn't affect the already launched go-routines.
// So instead we introduce the Pending API in a small commit in go-ethereum
// so we don't even start the miner there, and instead manage the intermediate state from here.
// In the same commit we also fire the TxPreEvent synchronously so the order is preserved,
// instead of using a go-routine.

type EthState struct {
	ethereum  *eth.Ethereum
	ethConfig *eth.Config

	mtx  sync.Mutex
	work workState // latest working state
}

type ChainError struct {
	ErrorCode   int    // Describes the kind of error
	Description string // Human readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e ChainError) Error() string {
	return e.Description
}

// chainError creates an RuleError given a set of arguments.
func chainError(c int, desc string) ChainError {
	return ChainError{ErrorCode: c, Description: desc}
}

func (es *EthState) State() (*state.StateDB, error) {
	return es.work.state, nil
}

// After NewEthState, call SetEthereum and SetEthConfig.
func NewEthState() *EthState {
	return &EthState{
		ethereum:  nil, // set with SetEthereum
		ethConfig: nil, // set with SetEthConfig
	}
}

func (es *EthState) SetEthereum(ethereum *eth.Ethereum) {
	es.ethereum = ethereum
}

func (es *EthState) SetEthConfig(ethConfig *eth.Config) {
	es.ethConfig = ethConfig
}

// Execute the transaction.
func (es *EthState) DeliverTx(tx *ethTypes.Transaction, appVersion uint64, txInfo ethTypes.TxInfo) abciTypes.ResponseDeliverTx {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	blockchain := es.ethereum.BlockChain()
	chainConfig := blockchain.Config()
	blockHash := common.Hash{}
	return es.work.deliverTx(blockchain, es.ethConfig, chainConfig, blockHash, tx, txInfo)
}

// Accumulate validator rewards.
func (es *EthState) AccumulateRewards(strategy *emtTypes.Strategy) {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	//cancel block rewards when blockversion >= 4
	if strategy.HFExpectedData.BlockVersion >= 4 {
		es.work.header.GasUsed = *es.work.totalUsedGas
	} else {
		es.work.accumulateRewards(strategy)
	}
}

// Commit and reset the work.
func (es *EthState) Commit() (common.Hash, error) {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	blockHash, err := es.work.commit(es.ethereum.BlockChain(), es.ethereum.ChainDb())
	if err != nil {
		return common.Hash{}, err
	}

	ws := &es.work
	err = es.resetWorkState(ws.header.Coinbase) //built for nextHeight, the coinbase in the header will later be overwritten in the next height
	if err != nil {
		return common.Hash{}, err
	}

	return blockHash, err
}

func (es *EthState) WorkState() workState {
	return es.work
}

func (es *EthState) ResetWorkState(receiver common.Address) error {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	return es.resetWorkState(receiver)
}

func (es *EthState) resetWorkState(receiver common.Address) error {

	blockchain := es.ethereum.BlockChain()
	state, err := blockchain.State()
	if err != nil {
		return err
	}

	currentBlock := blockchain.CurrentBlock()
	ethHeader := newBlockHeader(receiver, currentBlock)

	es.work = workState{
		header:       ethHeader,
		parent:       currentBlock,
		height:       blockchain.PendingBlock().Number().Int64(),
		state:        state,
		txIndex:      0,
		totalUsedGas: new(uint64),
		gp:           new(core.GasPool).AddGas(ethHeader.GasLimit),
	}
	return nil
}

func (es *EthState) UpdateHeaderWithTimeInfo(
	config *params.ChainConfig, parentTime uint64, numTx uint64) {

	es.mtx.Lock()
	defer es.mtx.Unlock()

	es.work.updateHeaderWithTimeInfo(config, parentTime, numTx)
}

func (es *EthState) UpdateHeaderCoinbase(
	coinbase common.Address) {

	es.mtx.Lock()
	defer es.mtx.Unlock()

	es.work.updateHeaderCoinbase(coinbase)
}

func (es *EthState) GasLimit() uint64 {
	return es.work.gp.Gas()
}

//----------------------------------------------------------------------
// Implements: miner.Pending API (our custom patch to go-ethereum)

// Return a new block and a copy of the state from the latest work.
// #unstable
func (es *EthState) Pending() (*ethTypes.Block, *state.StateDB) {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	return ethTypes.NewBlock(
		es.work.header,
		es.work.transactions,
		nil,
		es.work.receipts,
	), es.work.state.Copy()
}

//----------------------------------------------------------------------
//

// The work struct handles block processing.
// It's updated with each DeliverTx and reset on Commit.
type workState struct {
	header *ethTypes.Header
	parent *ethTypes.Block
	height int64
	state  *state.StateDB
	bstart time.Time //leilei add for gcproc

	txIndex      int
	transactions []*ethTypes.Transaction
	receipts     ethTypes.Receipts
	allLogs      []*ethTypes.Log

	totalUsedGas *uint64
	gp           *core.GasPool
}

func (ws *workState) State() *state.StateDB {
	return ws.state
}

func (ws workState) Height() int64 {
	return ws.height
}

// nolint: unparam
func (ws *workState) accumulateRewards(strategy *emtTypes.Strategy) {
	//ws.state.AddBalance(ws.header.Coinbase, ethash.FrontierBlockReward)
	log.Info(fmt.Sprintf("accumulateRewards LastVoteInfo %v", strategy.CurrentHeightValData.LastVoteInfo))
	minerBonus := strategy.CurrEpochValData.MinorBonus

	if strategy.CurrentHeightValData.Height <= 3588000 {
		minerBonus = big.NewInt(1)
		divisor := big.NewInt(1)
		// for 1% every year increment
		minerBonus.Div(strategy.CurrEpochValData.TotalBalance, divisor.Mul(big.NewInt(100), big.NewInt(365*24*60*60/5)))
	} else {
		ws.state.AddBalance(ws.CurrentHeader().Coinbase, minerBonus)
		//log.Info(fmt.Sprintf("proposer %v , Beneficiary address: %v, get money: %v",
		//	strategy.CurrentHeightValData.ProposerAddress, ws.CurrentHeader().Coinbase.String(), minerBonus))
	}

	weightSum := int64(0)
	for _, voteInfo := range strategy.CurrentHeightValData.LastVoteInfo {
		if voteInfo.SignedLastBlock {
			weightSum = weightSum + voteInfo.Validator.Power
		}
	}

	for i, voteInfo := range strategy.CurrentHeightValData.LastVoteInfo {
		if !voteInfo.SignedLastBlock || voteInfo.Validator.Power == 0 {
			continue
		}
		if voteInfo.Validator.Power < 0 {
			panic(fmt.Sprintf("Validator.Power < 0 %v", voteInfo))
		}
		bonusAverage := big.NewInt(1)
		bonusSpecify := big.NewInt(1)
		bonusSpecify.Mul(big.NewInt(voteInfo.Validator.Power), bonusAverage.
			Div(minerBonus, big.NewInt(int64(weightSum))))

		address := strings.ToUpper(hex.EncodeToString(
			strategy.CurrentHeightValData.LastVoteInfo[i].Validator.Address))
		var beneficiary common.Address
		if signer, ok := strategy.CurrEpochValData.PosTable.TmAddressToSignerMap[address]; ok {
			posItem, found := strategy.CurrEpochValData.PosTable.PosItemMap[signer]
			if found {
				beneficiary = posItem.Beneficiary
			} else { //the validator has just unbonded
				posItem, found := strategy.CurrEpochValData.PosTable.UnbondPosItemMap[signer]
				if found {
					beneficiary = posItem.Beneficiary
				} else {
					panic(fmt.Sprintf("address %v exist in TmAddressToSignerMap, but not found in either posItemMap or UnbondPosItemMap", signer))
				}
			}
		} else {
			panic(fmt.Sprintf("address %v not exist in TmAddressToSignerMap", address))
		}
		if strategy.HFExpectedData.BlockVersion >= 3 {
			ws.state.AddBalance(beneficiary, bonusSpecify)
		} else {
			ws.state.AddBalance(beneficiary, bonusAverage) //bug
		}

		//log.Info(fmt.Sprintf("validator %v , Beneficiary address: %v, get money: %v power: %v validator address: %v",
		//	strconv.Itoa(i+1), beneficiary.String(), bonusSpecify.String(),
		//	voteInfo.Validator.Power, address))
	}

	//This is no statistic data
	/*if strategy.HFExpectedData.StatisticsVersion == 0 {
		// upgrade success and run the new logic after upgrade height
		if strategy.HFExpectedData.BlockVersion-version.BeforeHardForkVersion == 1 &&
			strategy.HFExpectedData.Height >= version.NextHardForkHeight {
			log.Info("fix gas bonus bug")
		} else {
			//upgrade failed or before upgrade run the old logic
			log.Info("mock gas bug")
			ws.state.AddBalance(common.HexToAddress("8423328b8016fbe31938a461b5647de696bdbf71"), minerBonus)
		}
	} else {
		// upgrade based the statistic data
		if strategy.HFExpectedData.IsHarfForkPassed &&
			strategy.HFExpectedData.BlockVersion-version.BeforeHardForkVersion == 1 &&
			strategy.HFExpectedData.Height >= version.NextHardForkHeight {
			strategy.HFExpectedData.BlockVersion = 1
			log.Info("hard fork by statistic data")
		}
	}*/

	ws.header.GasUsed = *ws.totalUsedGas
}

// Runs ApplyTransaction against the ethereum blockchain, fetches any logs,
// and appends the tx, receipt, and logs.
func (ws *workState) deliverTx(blockchain *core.BlockChain, config *eth.Config,
	chainConfig *params.ChainConfig, blockHash common.Hash,
	tx *ethTypes.Transaction, txInfo ethTypes.TxInfo) abciTypes.ResponseDeliverTx {
	ws.state.Prepare(tx.Hash(), blockHash, ws.txIndex)
	var err error
	var msg core.Message
	var receipt *ethTypes.Receipt
	receipt, msg, _, err = core.ApplyTransactionWithInfo(
		chainConfig,
		blockchain,
		&ws.header.Coinbase, // defaults to address of the author of the header
		ws.gp,
		ws.state,
		ws.header,
		tx,
		txInfo,
		ws.totalUsedGas,
		vm.Config{EnablePreimageRecording: config.EnablePreimageRecording},
	)

	if err != nil {
		log.Error(fmt.Sprintf("Deliver Tx: err %v", err))
		return abciTypes.ResponseDeliverTx{Code: errorCode, Log: err.Error()}
	}
	log.Debug(fmt.Sprintf("Deliver Tx: from %X tx %v", msg.From(), tx))

	logs := ws.state.GetLogs(tx.Hash())

	ws.txIndex++

	// The slices are allocated in updateHeaderWithTimeInfo
	ws.transactions = append(ws.transactions, tx)
	ws.receipts = append(ws.receipts, receipt)
	ws.allLogs = append(ws.allLogs, logs...)

	return abciTypes.ResponseDeliverTx{Code: abciTypes.CodeTypeOK}
}

// Commit the ethereum state, update the header, make a new block and add it to
// the ethereum blockchain. The application root hash is the hash of the
// ethereum block.
func (ws *workState) commit(blockchain *core.BlockChain, db ethdb.Database) (common.Hash, error) {

	// Commit ethereum state and update the header.
	hashArray, err := ws.state.Commit(false) // XXX: ugh hardforks
	if err != nil {
		return common.Hash{}, err
	}

	ws.header.Root = hashArray

	for _, log := range ws.allLogs {
		log.BlockHash = hashArray
	}

	for _, r := range ws.receipts {
		for _, l := range r.Logs {
			l.BlockHash = hashArray
		}
	}

	// Create block object and compute final commit hash (hash of the ethereum
	// block).
	block := ethTypes.NewBlock(ws.header, ws.transactions, nil, ws.receipts)
	blockHash := block.Hash()

	log.Info(fmt.Sprintf("eth_state commit. block.header %v blockHash %X",
		block.Header(), blockHash))

	proctime := time.Since(ws.bstart)
	blockchain.AddGcproc(proctime)
	stat, err := blockchain.WriteBlockWithState(block, ws.receipts, ws.allLogs, ws.state, true)
	if err != nil {
		log.Error("Failed writing block to chain", "err", err)
		return common.Hash{}, err
	}
	// check if canon block and write transactions
	/*	var (
			events []interface{}
			//logs   = work.state.Logs()
		)
		events = append(events, core.ChainEvent{Block: block, Hash: block.Hash(), Logs: ws.allLogs})
		if stat == core.CanonStatTy {
			events = append(events, core.ChainHeadEvent{Block: block}) //此事件更新txPool
		} else {
			err = chainError(1, "WriteBlockWithState return stat not CanonStatTy")
			log.Error("stat not core.CanonStatTy", "workState", ws)
		}*/
	/*blockchain.mux.Post(core.NewMinedBlockEvent{Block: block})
	交易通过tendermint广播，此事件不用发
	*/
	//blockchain.PostChainEvents(events, ws.allLogs)
	// Save the block to disk.
	log.Info("Committing block", "stateHash", hashArray, "blockHash", blockHash, "stat", stat)
	/*	_, err = blockchain.InsertChain([]*ethTypes.Block{block})
		if err != nil {
			// log.Info("Error inserting ethereum block in chain", "err", err)
			return common.Hash{}, err
		}*/
	return blockHash, err
}

func (ws *workState) updateHeaderCoinbase(coinbase common.Address) {
	ws.header.Coinbase = coinbase
}

func (ws *workState) updateHeaderWithTimeInfo(
	config *params.ChainConfig, parentTime uint64, numTx uint64) {

	lastBlock := ws.parent
	parentHeader := &ethTypes.Header{
		Difficulty: lastBlock.Difficulty(),
		Number:     lastBlock.Number(),
		Time:       lastBlock.Time(),
	}
	ws.header.Time = parentTime
	ws.bstart = time.Now()
	ws.header.Difficulty = ethash.CalcDifficulty(config, parentTime, parentHeader)
	ws.transactions = make([]*ethTypes.Transaction, 0, numTx)
	ws.receipts = make([]*ethTypes.Receipt, 0, numTx)
	ws.allLogs = make([]*ethTypes.Log, 0, numTx)
}

//----------------------------------------------------------------------
func (ws *workState) CurrentHeader() *ethTypes.Header {
	return ws.header
}

// Create a new block header from the previous block.
func newBlockHeader(receiver common.Address, prevBlock *ethTypes.Block) *ethTypes.Header {
	return &ethTypes.Header{
		Number:     prevBlock.Number().Add(prevBlock.Number(), big.NewInt(1)),
		ParentHash: prevBlock.Hash(),
		GasLimit:   core.CalcGasLimit(prevBlock),
		Coinbase:   receiver,
	}
}
