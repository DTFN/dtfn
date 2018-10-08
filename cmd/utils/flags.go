// nolint=lll
package utils

import (
	"gopkg.in/urfave/cli.v1"
)

var (
	LogLevelFlag = cli.StringFlag{
		Name:  "logLevel",
		Value: "info",
		Usage: "log level for both ethermint and tendermint.",
	}
	// ----------------------------
	// ABCI Flags

	// TendermintAddrFlag is the address that ethermint will use to connect to the tendermint core node
	// #stable - 0.4.0
	TendermintAddrFlag = cli.StringFlag{
		Name:  "tendermint_addr",
		Value: "tcp://localhost:26657",
		Usage: "This is the address that ethermint will use to connect to the tendermint core node. Please provide a port.",
	}

	// ABCIAddrFlag is the address that ethermint will use to listen to incoming ABCI connections
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

	// ConfigFileFlag defines the path to a TOML config for go-ethereum
	// #unstable
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}

	// TargetGasLimitFlag defines gas limit of the Genesis block
	// #unstable
	TargetGasLimitFlag = cli.Uint64Flag{
		Name:  "targetgaslimit",
		Usage: "Target gas limit sets the artificial target gas floor for the blocks to mine",
		Value: GenesisTargetGasLimit.Uint64(),
	}

	// WithTendermintFlag asks to start Tendermint
	// `tendermint init` and `tendermint node` when `ethermint init`
	// and `ethermint` are invoked respectively.
	WithTendermintFlag = cli.BoolFlag{
		Name: "with-tendermint",
		Usage: "If set, it will invoke `tendermint init` and `tendermint node` " +
			"when `ethermint init` and `ethermint` are invoked respectively",
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
)
