package ethereum

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	abciTypes "github.com/tendermint/tendermint/abci/types"

	rpcClient "github.com/tendermint/tendermint/rpc/lib/client"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	emtTypes "github.com/DTFN/dtfn/types"
	mempl "github.com/tendermint/tendermint/mempool"
)

//----------------------------------------------------------------------
// Backend manages the underlying ethereum state for storage and processing,
// and maintains the connection to Tendermint for forwarding txs

// Backend handles the chain database and VM
// #stable - 0.4.0
type Backend struct {
	// backing ethereum structures
	ethereum  *eth.Ethereum
	ethConfig *eth.Config

	// txBroadcastLoop subscription
	txSub *event.TypeMuxSubscription

	// EthState
	es *EthState

	// client for forwarding txs to Tendermint
	client rpcClient.HTTPClient

	//leilei add.  Use mempool to forward txs directly
	memPool       mempl.Mempool
	currentTxInfo ethTypes.TxInfo
	cachedTxInfo  map[common.Hash]ethTypes.TxInfo
}

// NewBackend creates a new Backend
// #stable - 0.4.0
func NewBackend(node *Node, ethConfig *eth.Config,
	client rpcClient.HTTPClient) (*Backend, error) {

	// Create working ethereum state.
	es := NewEthState()

	// eth.New takes a ServiceContext for the EventMux, the AccountManager,
	// and some basic functions around the DataDir.
	ethereum, err := eth.New(&node.Node, ethConfig)
	if err != nil {
		return nil, err
	}
	ethereum.StopMining()

	es.SetEthereum(ethereum)
	es.SetEthConfig(ethConfig)

	// send special event to go-ethereum to switch homestead=true.
	currentBlock := ethereum.BlockChain().CurrentBlock()
	ethereum.EventMux().Post(core.ChainHeadEvent{currentBlock}) // nolint: vet, errcheck

	// We don't need PoW/Uncle validation.
	ethereum.BlockChain().SetValidator(core.NewBlockValidator(ethereum.BlockChain().Config(), ethereum.BlockChain(), ethash.NewFaker()))

	ethBackend := &Backend{
		ethereum:     ethereum,
		ethConfig:    ethConfig,
		es:           es,
		client:       client,
		cachedTxInfo: make(map[common.Hash]ethTypes.TxInfo),
	}
	return ethBackend, nil
}

// Ethereum returns the underlying the ethereum object.
// #stable
func (b *Backend) Ethereum() *eth.Ethereum {
	return b.ethereum
}

func (b *Backend) Es() *EthState {
	return b.es
}

// Config returns the eth.Config.
// #stable
func (b *Backend) Config() *eth.Config {
	return b.ethConfig
}

func (b *Backend) SetMemPool(memPool mempl.Mempool) {
	b.memPool = memPool
}

func (b *Backend) MemPool() mempl.Mempool {
	return b.memPool
}

func (b *Backend) CurrentTxInfo() ethTypes.TxInfo {
	return b.currentTxInfo
}

func (b *Backend) CachedTxInfo() map[common.Hash]ethTypes.TxInfo {
	return b.cachedTxInfo
}

//----------------------------------------------------------------------
// Handle block processing

// DeliverTx appends a transaction to the current block
// #stable
func (b *Backend) DeliverTx(tx *ethTypes.Transaction, appVerion uint64, txInfo ethTypes.TxInfo) abciTypes.ResponseDeliverTx {
	return b.es.DeliverTx(tx, appVerion, txInfo)
}

// AccumulateRewards accumulates the rewards based on the given strategy
// #unstable
func (b *Backend) AccumulateRewards(strategy *emtTypes.Strategy) {
	b.es.AccumulateRewards(strategy)
}

func (b *Backend) FetchCachedTxInfo(txHash common.Hash) (ethTypes.TxInfo, bool) {
	txInfo, ok := b.cachedTxInfo[txHash]
	return txInfo, ok
}

func (b *Backend) DeleteCachedTxInfo(txHash common.Hash) {
	delete(b.cachedTxInfo, txHash)
}

func (b *Backend) InsertCachedTxInfo(txHash common.Hash, txInfo ethTypes.TxInfo) {
	b.cachedTxInfo[txHash] = txInfo
}

// Commit finalises the current block
// #unstable
func (b *Backend) Commit() (common.Hash, error) {
	appHash, err := b.es.Commit()
	/*	if err!=nil{
		b.ethereum.TxPool().Loop()
	}*/
	return appHash, err
}

// InitEthState initializes the EthState
// #unstable
func (b *Backend) InitEthState(receiver common.Address) error {
	return b.es.ResetWorkState(receiver)
}

func (b *Backend) InitReceiver() string {
	return "0000000000000000000000000000000000000002" //will be overwritten by CurrentHeightValData.ProposerAddress
}

// UpdateHeaderWithTimeInfo uses the tendermint header to update the ethereum header
// #unstable
func (b *Backend) UpdateHeaderWithTimeInfo(tmHeader *abciTypes.Header) {
	b.es.UpdateHeaderWithTimeInfo(b.ethereum.BlockChain().Config(), uint64(tmHeader.Time.Unix()),
		uint64(tmHeader.GetNumTxs()))
}

// GasLimit returns the maximum gas per block
// #unstable
func (b *Backend) GasLimit() uint64 {
	return b.es.GasLimit()
}

//----------------------------------------------------------------------
// Implements: node.Service

// APIs returns the collection of RPC services the ethereum package offers.
// #stable - 0.4.0
func (b *Backend) APIs() []rpc.API {
	apis := b.Ethereum().APIs()
	retApis := []rpc.API{}
	for _, v := range apis {
		if v.Namespace == "net" {
			v.Service = NewNetRPCService(b.ethConfig.NetworkId)
		}
		if v.Namespace == "miner" {
			continue
		}
		if _, ok := v.Service.(*eth.PublicMinerAPI); ok {
			continue
		}
		retApis = append(retApis, v)
	}
	return retApis
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
// #stable
func (b *Backend) Start(_ *p2p.Server) error {
	go b.txBroadcastLoop()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
// #stable
func (b *Backend) Stop() error {
	b.txSub.Unsubscribe()
	b.ethereum.Stop() // nolint: errcheck
	return nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
// #stable
func (b *Backend) Protocols() []p2p.Protocol {
	return nil
}

//----------------------------------------------------------------------
// We need a block processor that just ignores PoW and uncles and so on

// NullBlockProcessor does not validate anything
// #unstable
type NullBlockProcessor struct{}

// ValidateBody does not validate anything
// #unstable
func (v *NullBlockProcessor) ValidateBody(block *ethTypes.Block) error {
	return nil
}

// ValidateState does not validate anything
// #unstable
func (v *NullBlockProcessor) ValidateState(block, parent *ethTypes.Block, state *state.StateDB, receipts ethTypes.Receipts, usedGas uint64) error {
	return nil
}

type TxFrom struct {
	TxHash common.Hash
	From   common.Address
}
