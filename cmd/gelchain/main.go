package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/params"

	"github.com/green-element-chain/gelchain/cmd/utils"
	"github.com/green-element-chain/gelchain/version"
)

var (
	// The app that holds all commands and flags.
	app = ethUtils.NewApp(version.Version, "the gelchain command line interface")
	// flags that configure the go-ethereum node
	nodeFlags = []cli.Flag{
		ethUtils.DataDirFlag,
		ethUtils.KeyStoreDirFlag,
		ethUtils.NoUSBFlag,
		ethUtils.FakePoWFlag,
		// Performance tuning
		ethUtils.CacheFlag,
		ethUtils.TrieCacheGenFlag,
		ethUtils.GCModeFlag,
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
		// Gas Price
		ethUtils.GasPriceFlag,
		//network setting
		ethUtils.LightPeersFlag,
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
	}

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, ethermintFlags...)
	app.Flags = append(app.Flags, tendermintFlags...)

	app.Before = func(ctx *cli.Context) error {
		if err := utils.Setup(ctx); err != nil {
			return err
		}

		ethUtils.SetupNetwork(ctx)

		return nil
	}
}

func versionCmd(ctx *cli.Context) error {
	fmt.Println("gelchain: ", version.Version)
	fmt.Println("go-ethereum: ", params.Version)
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}