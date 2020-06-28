package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/params"

	"github.com/DTFN/dtfn/cmd/utils"
	"github.com/DTFN/dtfn/version"
	"time"
)

var (
	// The app that holds all commands and flags.
	app = ethUtils.NewApp(version.Version, time.Now().String(), "the gelchain command line interface")
	// flags that configure the go-ethereum node
	nodeFlags = []cli.Flag{
		ethUtils.DataDirFlag,
		ethUtils.KeyStoreDirFlag,
		ethUtils.NoUSBFlag,
		ethUtils.FakePoWFlag,
		// Performance tuning
		ethUtils.CacheFlag,
		ethUtils.GCModeFlag,
		utils.TrieTimeLimitFlag,
		// Account settings
		ethUtils.UnlockedAccountFlag,
		ethUtils.PasswordFileFlag,
		ethUtils.VMEnableDebugFlag,
		// Logging and debug settings
		ethUtils.NoCompactionFlag,
		// Gas price oracle settings
		ethUtils.GpoBlocksFlag,
		ethUtils.GpoPercentileFlag,
		utils.TargetGasLimitFlag,
		utils.TxpoolThreshold,
		utils.TxpoolPriceLimit,
		utils.LRUCacheSize,
		ethUtils.InsecureUnlockAllowedFlag,
		ethUtils.MaxPeersFlag,
	}

	rpcFlags = []cli.Flag{
		ethUtils.RPCEnabledFlag,
		ethUtils.RPCListenAddrFlag,
		ethUtils.RPCPortFlag,
		ethUtils.RPCCORSDomainFlag,
		ethUtils.RPCVirtualHostsFlag,
		ethUtils.RPCApiFlag,
		ethUtils.IPCDisabledFlag,
		ethUtils.WSEnabledFlag,
		ethUtils.WSListenAddrFlag,
		ethUtils.WSPortFlag,
		ethUtils.WSApiFlag,
		ethUtils.WSAllowedOriginsFlag,
	}

	// flags that configure the ABCI app
	ethermintFlags = []cli.Flag{
		utils.TendermintAddrFlag,
		utils.ABCIAddrFlag,
		utils.ABCIProtocolFlag,
		utils.VerbosityFlag,
		utils.ConfigFileFlag,
		utils.WithTendermintFlag,
		utils.VersionConfigFile,
		utils.VersionConfigTypeFlag,
		//log level
		utils.LogLevelFlag,
	}

	// flags that configure the ABCI app
	tendermintFlags = []cli.Flag{
		utils.PexReactor,
		utils.PrivValidatorListenAddr,
		utils.PrivValidator,
		utils.FastSync,
		utils.PersistentPeers,
		utils.AddrBook,
		utils.RoutabilityStrict,
		utils.PrivatePeerIDs,
		utils.TendermintP2PListenAddress,
		utils.TendermintP2PExternalAddress,
		utils.MempoolBroadcastFlag,
		utils.TmConsEmptyBlock,
		utils.TmConsEBlockInteval,
		utils.TmConsNeedProofBlock,
		utils.TmConsProposeTimeout,
		utils.TmInitialEthAccount,
		utils.TmBlsSelectStrategy,
		utils.TestNetHostnamePrefix,
		utils.TestNetNodeDir,
		utils.TestNetNVals,
		utils.TestNetOutput,
		utils.TestNetP2PPort,
		utils.TestNetpOpulatePersistentPeers,
		utils.TestnetStartingIPAddress,
		utils.TestNetVals,
		utils.MaxInPeers,
		utils.MaxOutPeers,
		utils.RollbackHeight,
		utils.RollbackFlag,
		utils.SelectCount,
		utils.SelectBlockNumber,
		utils.SelectStrategy,
		utils.MempoolSize,
		utils.MempoolThreshold,
		utils.MempoolHeightThreshold,
		utils.FlowControlFlag,
		utils.FlowControlMaxSleepTime,
	}
)

func init() {
	app.Action = ethermintCmd
	app.HideVersion = true
	app.Commands = []cli.Command{
		{
			Action:      initCmd,
			Name:        "init",
			Usage:       "init genesis.json",
			Description: "Initialize the files",
		},
		{
			Action:      versionCmd,
			Name:        "version",
			Usage:       "",
			Description: "Print the version",
		},
		{
			Action: resetCmd,
			Name:   "unsafe_reset_all",
			Usage:  "(unsafe) Remove gelchain database",
		},
		{
			Action: testnetCmd,
			Name:   "testnet",
			Usage:  "generate ,the test config file",
		},
	}

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, ethermintFlags...)
	app.Flags = append(app.Flags, tendermintFlags...)

	app.Before = func(ctx *cli.Context) error {
		if err := utils.Setup(ctx); err != nil {
			return err
		}
		//ethUtils.SetupNetwork(ctx)

		return nil
	}
}

func versionCmd(ctx *cli.Context) error {
	fmt.Println("dtfn: ", version.Version)
	fmt.Println("go-ethereum: ", params.Version)
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
