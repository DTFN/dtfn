// nolint=lll
package utils

import (
	"gopkg.in/urfave/cli.v1"
)

var (
	LogLevelFlag = cli.StringFlag{
		Name:  "logLevel",
		Value: "info",
		Usage: "log level for both gelchain and tendermint.",
	}
	// ----------------------------
	// ABCI Flags

	// TendermintAddrFlag is the address that gelchain will use to connect to the tendermint core node
	// #stable - 0.4.0
	TendermintAddrFlag = cli.StringFlag{
		Name:  "tendermint_addr",
		Value: "tcp://localhost:26657",
		Usage: "This is the address that gelchain will use to connect to the tendermint core node. Please provide a port.",
	}

	// ABCIAddrFlag is the address that gelchain will use to listen to incoming ABCI connections
	// #stable - 0.4.0
	ABCIAddrFlag = cli.StringFlag{
		Name:  "abci_laddr",
		Value: "tcp://0.0.0.0:26658",
		Usage: "This is the address that the ABCI server will use to listen to incoming connection from tendermint core.",
	}

	// ABCIProtocolFlag defines whether GRPC or SOCKET should be used for the ABCI connections
	// #stable - 0.4.0
	ABCIProtocolFlag = cli.StringFlag{
		Name:  "abci_protocol",
		Value: "socket",
		Usage: "socket | grpc",
	}

	// VerbosityFlag defines the verbosity of the logging
	// #unstable
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Value: 3,
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=core, 5=debug, 6=detail",
	}

	// TrieTimeLimitFlag defines how long would a memory trie flush into database
	// #unstable
	TrieTimeLimitFlag = cli.IntFlag{
		Name:  "trie_time_limit",
		Value: 60, //Second
		Usage: "how long would a memory trie flush into database",
	}

	// ConfigFileFlag defines the path to a TOML config for go-ethereum
	// #unstable
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}

	// TargetGasLimitFlag defines gas limit of the Genesis block
	// #unstable
	TargetGasLimitFlag = cli.Uint64Flag{
		Name:  "target_gas_limit",
		Usage: "Target gas limit sets the artificial target gas floor for the blocks to mine",
		Value: GenesisTargetGasLimit.Uint64(),
	}

	// WithTendermintFlag asks to start Tendermint
	// `tendermint init` and `tendermint node` when `gelchain init`
	// and `gelchain` are invoked respectively.
	WithTendermintFlag = cli.BoolFlag{
		Name: "with-tendermint",
		Usage: "If set, it will invoke `tendermint init` and `tendermint node` " +
			"when `gelchain init` and `gelchain` are invoked respectively",
	}

	//=======================================tendermint flags====================
	PrivValidatorListenAddr = cli.StringFlag{
		Name:  "priv_validator_laddr",
		Usage: "TCP or UNIX socket address for Tendermint to listen on for connections from an external PrivValidator process",
		Value: "",
	}

	PrivValidator = cli.StringFlag{
		Name:  "priv_validator_file",
		Usage: "Path to the JSON file containing the private key to use as a validator in the consensus protocol",
		Value: "",
	}

	FastSync = cli.BoolFlag{
		Name:  "fast_sync",
		Usage: "If this node is many blocks behind the tip of the chain, FastSync allows them to catchup quickly by downloading blocks in paralleland verifying their commits",
	}

	PersistentPeers = cli.StringFlag{
		Name:  "persistent_peers",
		Usage: "Comma separated list of nodes to keep persistent connections to. Do not add private peers to this list if you don't want them advertised",
		Value: "",
	}

	AddrBook = cli.StringFlag{
		Name:  "addr_book_file",
		Usage: "Path to address book",
		Value: "",
	}

	RoutabilityStrict = cli.BoolFlag{
		Name:  "routable_strict",
		Usage: "routabilityStrict property of address book.If set,will check the address not local or LAN.",
	}

	PrivatePeerIDs = cli.StringFlag{
		Name:  "private_peer_ids",
		Usage: "Comma separated list of peer IDs to keep private (will not be gossiped to other peers)",
		Value: "",
	}

	PexReactor = cli.BoolFlag{
		Name:  "pex",
		Usage: "Set true to enable the peer-exchange reactor",
	}
	// Comma separated list of peer IDs to keep private (will not be gossiped to other peers)

	TendermintP2PListenAddress = cli.StringFlag{
		Name:  "tendermint_p2paddr",
		Value: "",
		Usage: "This is the address that tendermint will use to connect other tendermint port.",
	}

	TendermintP2PExternalAddress = cli.StringFlag{
		Name:  "tm_external_addr",
		Value: "",
		Usage: "Address to advertise to peers for them to dial.If empty, will use the same port as the laddr",
	}

	MempoolBroadcastFlag = cli.BoolFlag{
		Name:  "broadcast_tx",
		Usage: "If set, mempool will broadcast tx",
	}

	TmConsEmptyBlock = cli.BoolFlag{
		Name:  "tm_cons_emptyblock",
		Usage: "EmptyBlocks mode",
	}

	TmConsEBlockInteval = cli.Uint64Flag{
		Name:  "tm_cons_eb_inteval",
		Usage: "possible interval between empty blocks in seconds",
	}

	TmConsProposeTimeout = cli.Uint64Flag{
		Name:  "propose_timeout",
		Usage: "propose timeout in seconds",
	}

	TmConsNeedProofBlock = cli.BoolFlag{
		Name:  "need_proof_block",
		Usage: "whether to need proof block",
	}

	TmInitialEthAccount = cli.StringFlag{
		Name:  "initial_eth_account",
		Usage: "initial_eth_account to config the initial node",
	}

	TmBlsSelectStrategy = cli.BoolFlag{
		Name:  "bls_select_strategy",
		Usage: "specify select strategy for bls",
	}

	TestNetVals = cli.IntFlag{
		Name:  "v",
		Value: 4,
		Usage: "Number of validators to initialize the testnet with",
	}

	TestNetNVals = cli.IntFlag{
		Name:  "n",
		Value: 0,
		Usage: "Number of non-validators to initialize the testnet with",
	}

	TestNetP2PPort = cli.IntFlag{
		Name:  "p2p-port",
		Value: 26656,
		Usage: "P2P Port",
	}

	TestNetpOpulatePersistentPeers = cli.BoolFlag{
		Name:   "populate-persistent-peers",
		Hidden: true,
		Usage:  "Update config of each node with the list of persistent peers build using either hostname-prefix or starting-ip-address",
	}

	TestNetOutput = cli.StringFlag{
		Name:  "o",
		Value: "./mytestnet",
		Usage: "Directory to store initialization data for the testnet",
	}

	TestNetNodeDir = cli.StringFlag{
		Name:  "node-dir-prefix",
		Value: "node",
		Usage: "Prefix the directory name for each node with (node results in node0, node1, ...)",
	}

	TestNetHostnamePrefix = cli.StringFlag{
		Name:  "hostname-prefix",
		Value: "node",
		Usage: "Hostname prefix (node results in persistent peers list ID0@node0:26656, ID1@node1:26656, ...)",
	}

	TestnetStartingIPAddress = cli.StringFlag{
		Name:  "starting-ip-address",
		Value: "",
		Usage: "Starting IP address (192.168.0.1 results in persistent peers list ID0@192.168.0.1:26656, ID1@192.168.0.2:26656, ...)",
	}

	RollbackHeight = cli.Int64Flag{
		Name:  "rollback_height",
		Value: 200,
		Usage: "the height which want to rollback",
	}

	RollbackFlag = cli.BoolFlag{
		Name:  "rollback_flag",
		Usage: "whether or not rollback",
	}

	SelectCount = cli.IntFlag{
		Name:  "select_count",
		Value: 7,
		Usage: "how many different validators are selected each height",
	}

	MaxInPeers = cli.IntFlag{
		Name:  "max_in_peers",
		Value: 29,
		Usage: "max inbound peers allowed",
	}

	MaxOutPeers = cli.IntFlag{
		Name:  "max_out_peers",
		Value: 11,
		Usage: "max outbound peers allowed",
	}

	MempoolSize = cli.IntFlag{
		Name:  "mempool_size",
		Value: 50000,
		Usage: "the size of tendermint mempool",
	}

	MempoolThreshold = cli.IntFlag{
		Name:  "mempool_threshold",
		Value: 25000,
		Usage: "the threshold of tendermint mempool for flow control",
	}

	FlowControlFlag = cli.BoolFlag{
		Name:  "flow_control",
		Usage: "if flow control, receiving tx and broadcasting tx get slower when txs over half the size of mempool",
	}

	FlowControlMaxSleepTime = cli.IntFlag{
		Name:  "flow_control_sleep",
		Value: 6,
		Usage: "max sleep time for flow control, seconds",
	}
)
