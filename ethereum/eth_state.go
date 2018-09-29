package ethereum

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	//"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"

	abciTypes "github.com/tendermint/tendermint/abci/types"

	"encoding/hex"
	"github.com/ethereum/go-ethereum/log"
	emtTypes "github.com/tendermint/ethermint/types"
	"strings"
	"time"
	"github.com/ethereum/go-ethereum/core/blacklist"
	"github.com/tendermint/tendermint/crypto"
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
func (es *EthState) DeliverTx(tx *ethTypes.Transaction, address *common.Address) (abciTypes.ResponseDeliverTx, *Wrap) {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	blockchain := es.ethereum.BlockChain()
	chainConfig := es.ethereum.ApiBackend.ChainConfig()
	blockHash := common.Hash{}
	return es.work.deliverTx(blockchain, es.ethConfig, chainConfig, blockHash, tx, address)
}

// Accumulate validator rewards.
func (es *EthState) AccumulateRewards(strategy *emtTypes.Strategy) {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	es.work.accumulateRewards(strategy)
}

// Commit and reset the work.
func (es *EthState) Commit(receiver common.Address) (common.Hash, error) {
	es.mtx.Lock()
	defer es.mtx.Unlock()

	blockHash, err := es.work.commit(es.ethereum.BlockChain(), es.ethereum.ChainDb())
	if err != nil {
		return common.Hash{}, err
	}

	err = es.resetWorkState(receiver)
	if err != nil {
		return common.Hash{}, err
	}

	return blockHash, err
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
	state  *state.StateDB
	bstart time.Time //leilei add for gcproc

	txIndex      int
	transactions []*ethTypes.Transaction
	receipts     ethTypes.Receipts
	allLogs      []*ethTypes.Log

	totalUsedGas *uint64
	gp           *core.GasPool
}

type Wrap struct {
	T           string
	Signer      common.Address
	Balance     *big.Int
	Beneficiary common.Address
	Pubkey      crypto.PubKey
}

func (ws *workState) State() *state.StateDB {
	return ws.state
}

// nolint: unparam
func (ws *workState) accumulateRewards(strategy *emtTypes.Strategy) {
	//ws.state.AddBalance(ws.header.Coinbase, ethash.FrontierBlockReward)
	//todo:后续要获取到块的validators列表根据voting power按比例分配收益

	for i := 0; i < len(strategy.GetUpdatedValidators()); i++ {
		address := strings.ToLower(hex.EncodeToString(strategy.GetUpdatedValidators()[i].Address))
		ws.state.AddBalance(strategy.AccountMapList.MapList[address].Beneficiary, big.NewInt(1000000000000000000))
	}
	ws.header.GasUsed = *ws.totalUsedGas
}

// Runs ApplyTransaction against the ethereum blockchain, fetches any logs,
// and appends the tx, receipt, and logs.
func (ws *workState) deliverTx(blockchain *core.BlockChain, config *eth.Config,
	chainConfig *params.ChainConfig, blockHash common.Hash,
	tx *ethTypes.Transaction, address *common.Address) (abciTypes.ResponseDeliverTx, *Wrap) {
	log.Info("to:" + tx.To().Hex())
	ws.state.Prepare(tx.Hash(), blockHash, ws.txIndex)
	receipt, msg, _, err := core.ApplyTransactionFacade(
		chainConfig,
		blockchain,
		address, // defaults to address of the author of the header
		ws.gp,
		ws.state,
		ws.header,
		tx,
		ws.totalUsedGas,
		vm.Config{EnablePreimageRecording: config.EnablePreimageRecording},
	)

	if err != nil {
		return abciTypes.ResponseDeliverTx{Code: errorCode, Log: err.Error()}, &Wrap{}
	}
	log.Info("from:" + msg.From().Hex())
	wrap := handleTx(ws.state, msg)
	log.Info("handled tx")

	logs := ws.state.GetLogs(tx.Hash())

	ws.txIndex++

	// The slices are allocated in updateHeaderWithTimeInfo
	ws.transactions = append(ws.transactions, tx)
	ws.receipts = append(ws.receipts, receipt)
	ws.allLogs = append(ws.allLogs, logs...)

	return abciTypes.ResponseDeliverTx{Code: abciTypes.CodeTypeOK}, wrap
}

func handleTx(statedb *state.StateDB, msg core.Message) *Wrap {
	log.Info("handleTx, to:" + msg.To().Hex())
	if msg.To() != nil {
		if blacklist.IsLockTx(*msg.To()) {
			blacklist.BlacklistDB.Add(msg.From())
			data, _ := MarshalTxData(string(msg.Data()))
			balance := statedb.GetBalance(msg.From())
			return &Wrap{
				t:           "upsert",
				signer:      msg.From(),
				balance:     balance,
				beneficiary: common.HexToAddress(data.Beneficiary),
				pubkey:      data.Pv.PubKey,
			}
		} else if blacklist.IsUnlockTx(*msg.To()) {
			blacklist.BlacklistDB.Remove(msg.From())
			return &Wrap{
				t:      "remove",
				signer: msg.From(),
			}
		}
	}
	return &Wrap{}
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

	proctime := time.Since(ws.bstart)
	blockchain.AddGcproc(proctime)
	stat, err := blockchain.WriteBlockWithState(block, ws.receipts, ws.state)
	if err != nil {
		log.Error("Failed writing block to chain", "err", err)
		return common.Hash{}, err
	}
	// check if canon block and write transactions
	var (
		events []interface{}
		//logs   = work.state.Logs()
	)
	events = append(events, core.ChainEvent{Block: block, Hash: block.Hash(), Logs: ws.allLogs})
	if stat == core.CanonStatTy {
		events = append(events, core.ChainHeadEvent{Block: block}) //此事件更新txPool
	} else {
		err = chainError(1, "WriteBlockWithState return stat not CanonStatTy")
		log.Error("stat not core.CanonStatTy", "workState", ws)
	}
	/*blockchain.mux.Post(core.NewMinedBlockEvent{Block: block})
	交易通过tendermint广播，此事件不用发
	*/
	blockchain.PostChainEvents(events, ws.allLogs)
	// Save the block to disk.
	// log.Info("Committing block", "stateHash", hashArray, "blockHash", blockHash)
	/*	_, err = blockchain.InsertChain([]*ethTypes.Block{block})
		if err != nil {
			// log.Info("Error inserting ethereum block in chain", "err", err)
			return common.Hash{}, err
		}*/
	return blockHash, err
}

func (ws *workState) updateHeaderWithTimeInfo(
	config *params.ChainConfig, parentTime uint64, numTx uint64) {

	lastBlock := ws.parent
	parentHeader := &ethTypes.Header{
		Difficulty: lastBlock.Difficulty(),
		Number:     lastBlock.Number(),
		Time:       lastBlock.Time(),
	}
	ws.header.Time = new(big.Int).SetUint64(parentTime)
	ws.bstart = time.Now()
	ws.header.Difficulty = ethash.CalcDifficulty(config, parentTime, parentHeader)
	ws.transactions = make([]*ethTypes.Transaction, 0, numTx)
	ws.receipts = make([]*ethTypes.Receipt, 0, numTx)
	ws.allLogs = make([]*ethTypes.Log, 0, numTx)
}

//----------------------------------------------------------------------

// Create a new block header from the previous block.
func newBlockHeader(receiver common.Address, prevBlock *ethTypes.Block) *ethTypes.Header {
	return &ethTypes.Header{
		Number:     prevBlock.Number().Add(prevBlock.Number(), big.NewInt(1)),
		ParentHash: prevBlock.Hash(),
		GasLimit:   core.CalcGasLimit(prevBlock),
		Coinbase:   receiver,
	}
}
