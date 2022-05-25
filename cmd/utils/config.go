package utils

import (
	"math/big"
	"os"

	"gopkg.in/urfave/cli.v1"

	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/DTFN/dtfn/ethereum"

	"time"

	rpcClient "github.com/tendermint/tendermint/rpc/lib/client"
)

const (
	// Client identifier to advertise over the network
	clientIdentifier = "gelchain"
	// Environment variable for home dir
	emHome = "EMHOME"
)

var (
	// GenesisTargetGasLimit is the target gas limit of the Genesis block.
	// #unstable
	GenesisTargetGasLimit = big.NewInt(100000000)
)

type ethstatsConfig struct {
	URL string `toml:",omitempty"`
}

type gethConfig struct {
	Eth      eth.Config
	Node     node.Config
	Ethstats ethstatsConfig
}

// MakeFullNode creates a full go-ethereum node
// #unstable
func MakeFullNode(ctx *cli.Context) (*ethereum.Node, *ethereum.Backend) {
	stack, cfg := makeConfigNode(ctx)

	var backend *ethereum.Backend
	var err error
	tendermintLAddr := ctx.GlobalString(TendermintAddrFlag.Name)
	if backend, err = ethereum.NewBackend(stack, &cfg.Eth, rpcClient.NewURIClient(tendermintLAddr)); err != nil {
		ethUtils.Fatalf("Failed to ethereum.NewBackend: %v", err)
	}

	return stack, backend
}

// MakeMigrationNode creates a node to create snapshot
func MakeMigrationNode(ctx *cli.Context) (*ethereum.Node, *ethereum.Backend) {
	stack, cfg := makeConfigNode(ctx)

	backend, err := ethereum.NewBackend(stack, &cfg.Eth, nil)
	if err != nil {
		ethUtils.Fatalf("Failed to create archive read backend: %v", err)
	}

	return stack, backend
}

// MakeStandaloneNode creates a node to read blockchain data.
// There should no any inbound/outbound connection, no tendermint core.
// Only ipc is allowed for query.
func MakeStandaloneNode(ctx *cli.Context) (*ethereum.Node, *ethereum.Backend) {
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,
		Node: DefaultNodeConfig(),
	}
	ethUtils.SetNodeConfig(ctx, &cfg.Node)

	// override config, it's actually like --dev.
	cfg.Node.P2P.MaxPeers = 0
	cfg.Node.P2P.ListenAddr = ""
	cfg.Node.P2P.NoDiscovery = true
	cfg.Node.P2P.DiscoveryV5 = false
	cfg.Node.HTTPHost = ""
	cfg.Node.WSHost = ""

	stack, err := ethereum.New(&cfg.Node)
	if err != nil {
		ethUtils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	SetEthermintEthConfig(ctx, &cfg.Eth)
	ethUtils.SetEthConfig(ctx, &stack.Node, &cfg.Eth)

	backend, err := ethereum.NewBackend(stack, &cfg.Eth, nil)
	if err != nil {
		ethUtils.Fatalf("Failed to create archive read backend: %v", err)
	}

	return stack, backend
}

func makeConfigNode(ctx *cli.Context) (*ethereum.Node, gethConfig) {
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,
		Node: DefaultNodeConfig(),
	}

	ethUtils.SetNodeConfig(ctx, &cfg.Node)
	SetEthermintNodeConfig(&cfg.Node)
	stack, err := ethereum.New(&cfg.Node)
	if err != nil {
		ethUtils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	SetEthermintEthConfig(ctx, &cfg.Eth)
	ethUtils.SetEthConfig(ctx, &stack.Node, &cfg.Eth)

	return stack, cfg
}

// DefaultNodeConfig returns the default configuration for a go-ethereum node
// #unstable
func DefaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.Version
	cfg.HTTPModules = append(cfg.HTTPModules, "eth")
	cfg.WSModules = append(cfg.WSModules, "eth")
	cfg.IPCPath = "geth.ipc"

	emHome := os.Getenv(emHome)
	if emHome != "" {
		cfg.DataDir = emHome
	}

	return cfg
}

// SetEthermintNodeConfig takes a node configuration and applies dtfn specific configuration
// #unstable
func SetEthermintNodeConfig(cfg *node.Config) {
	cfg.P2P.MaxPeers = 0
	cfg.P2P.NoDiscovery = true
}

// SetEthermintEthConfig takes a ethereum configuration and applies dtfn specific configuration
// #unstable
func SetEthermintEthConfig(ctx *cli.Context, cfg *eth.Config) {
	/*cfg.MaxPeers = 0
	cfg.PowFake = true*/
	cfg.Ethash.PowMode = ethash.ModeNil
	cfg.Miner.GasCeil = 0 //disable the miner
	trieTimeLimit := ctx.GlobalInt(TrieTimeLimitFlag.Name)
	if trieTimeLimit > 0 {
		trieTimeout := int64(trieTimeLimit) * int64(time.Second)
		cfg.TrieTimeout = time.Duration(trieTimeout)
	}
	if ctx.GlobalBool(FlowControlFlag.Name) {
		cfg.TxPool.MempoolSize = uint64(ctx.GlobalInt(MempoolSize.Name))
		cfg.TxPool.MaxFlowLimitSleepTime = time.Duration(ctx.GlobalInt(FlowControlMaxSleepTime.Name)) * time.Second
		cfg.TxPool.FlowLimitThreshold = uint64(ctx.GlobalInt(TxpoolThreshold.Name))
	} else {
		cfg.TxPool.MempoolSize = uint64(0)
	}
	txpoolPriceLimit := ctx.GlobalInt(TxpoolPriceLimit.Name)
	cfg.TxPool.PriceLimit = uint64(txpoolPriceLimit)

	cacheSize := ctx.GlobalInt(LRUCacheSize.Name)
	if cacheSize > 0 {
		cfg.TxPool.LRUCacheSize = cacheSize
	}
}

// MakeDataDir retrieves the currently requested data directory
// #unstable
func MakeDataDir(ctx *cli.Context) string {
	path := node.DefaultDataDir()

	emHome := os.Getenv(emHome)
	if emHome != "" {
		path = emHome
	}

	if ctx.GlobalIsSet(ethUtils.DataDirFlag.Name) {
		path = ctx.GlobalString(ethUtils.DataDirFlag.Name)
	}

	if path == "" {
		ethUtils.Fatalf("Cannot determine default data directory, please set manually (--datadir)")
	}

	return path
}
