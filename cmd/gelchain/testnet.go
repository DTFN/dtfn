package main

import (
	"fmt"
	emtUtils "github.com/green-element-chain/gelchain/cmd/utils"
	"github.com/tendermint/tendermint/cmd/tendermint/commands"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/consensus"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
	"gopkg.in/urfave/cli.v1"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	nValidators    int
	nNonValidators int
	outputDir      string
	nodeDirPrefix  string

	populatePersistentPeers bool
	hostnamePrefix          string
	startingIPAddress       string
	p2pPort                 int

	rollbackHeight int
	rollbackFlag   bool
)

const (
	nodeDirPerm = 0755
)

func testnetCmd(ctx *cli.Context) error {
	config := cfg.DefaultConfig()

	nValiString := ctx.Args().First()
	nVali, err := strconv.Atoi(nValiString)
	if len(nValiString) != 0 && err == nil {
		nValidators = nVali
	} else {
		nValidators = ctx.GlobalInt(emtUtils.TestNetVals.Name)
	}

	nNonValidators = ctx.GlobalInt(emtUtils.TestNetNVals.Name)
	p2pPort = ctx.GlobalInt(emtUtils.TestNetP2PPort.Name)
	populatePersistentPeers = ctx.GlobalBool(emtUtils.TestNetpOpulatePersistentPeers.Name)

	outputDir = ctx.GlobalString(emtUtils.TestNetOutput.Name)
	nodeDirPrefix = ctx.GlobalString(emtUtils.TestNetNodeDir.Name)
	hostnamePrefix = ctx.GlobalString(emtUtils.TestNetHostnamePrefix.Name)
	startingIPAddress = ctx.GlobalString(emtUtils.TestnetStartingIPAddress.Name)

	rollbackHeight = ctx.GlobalInt(emtUtils.RollbackHeight.Name)
	rollbackFlag = ctx.GlobalBool(emtUtils.RollbackFlag.Name)

	genVals := make([]types.GenesisValidator, nValidators)

	for i := 0; i < nValidators; i++ {
		nodeDirName := fmt.Sprintf("%s%d", nodeDirPrefix, i)
		nodeDir := filepath.Join(outputDir, nodeDirName)
		config.SetRoot(nodeDir)

		err := os.MkdirAll(filepath.Join(nodeDir, "config"), nodeDirPerm)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}
		err = os.MkdirAll(filepath.Join(nodeDir, "data"), nodeDirPerm)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}

		commands.InitFilesWithConfig(config)

		oldPrivVal := config.OldPrivValidatorFile()
		newPrivValKey := config.PrivValidatorKeyFile()
		newPrivValState := config.PrivValidatorStateFile()
		if _, err := os.Stat(oldPrivVal); !os.IsNotExist(err) {
			oldPV, err := privval.LoadOldFilePV(oldPrivVal)
			if err != nil {
				return fmt.Errorf("error reading OldPrivValidator from %v: %v\n", oldPrivVal, err)
			}
			fmt.Println("Upgrading PrivValidator file",
				"old", oldPrivVal,
				"newKey", newPrivValKey,
				"newState", newPrivValState,
			)
			oldPV.Upgrade(newPrivValKey, newPrivValState)
		}
		pv:=privval.LoadOrGenFilePV(newPrivValKey, newPrivValState)
		blsState := consensus.LoadFileBS(filepath.Join(nodeDir, config.BaseConfig.BlsState))
		blsPubK := types.BLSPubKey{
			Type:    "Secp256k1",
			Address: pv.GetPubKey().Address().String(),
			Value:   blsState.GetPubKPKE(),
		}
		genVals[i] = types.GenesisValidator{
			PubKey:    pv.GetPubKey(),
			BlsPubKey: blsPubK,
			Power:     1,
			Name:      nodeDirName,
		}
	}

	for i := 0; i < nNonValidators; i++ {
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("%s%d", nodeDirPrefix, i+nValidators))
		config.SetRoot(nodeDir)

		err := os.MkdirAll(filepath.Join(nodeDir, "config"), nodeDirPerm)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}

		commands.InitFilesWithConfig(config)
	}

	// Generate genesis doc from generated validators
	genDoc := &types.GenesisDoc{
		GenesisTime: time.Now(),
		ChainID:     "chain-" + cmn.RandStr(6),
		Validators:  genVals,
	}

	// Write genesis file.
	for i := 0; i < nValidators+nNonValidators; i++ {
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("%s%d", nodeDirPrefix, i))
		if err := genDoc.SaveAs(filepath.Join(nodeDir, config.BaseConfig.Genesis)); err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}
	}

	if populatePersistentPeers {
		err := populatePersistentPeersInConfigAndWriteIt(config)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}
	}

	fmt.Printf("Successfully initialized %v node directories\n", nValidators+nNonValidators)
	return nil
}

func hostnameOrIP(i int) string {
	if startingIPAddress != "" {
		ip := net.ParseIP(startingIPAddress)
		ip = ip.To4()
		if ip == nil {
			fmt.Printf("%v: non ipv4 address\n", startingIPAddress)
			os.Exit(1)
		}

		for j := 0; j < i; j++ {
			ip[3]++
		}
		return ip.String()
	}

	return fmt.Sprintf("%s%d", hostnamePrefix, i)
}

func populatePersistentPeersInConfigAndWriteIt(config *cfg.Config) error {
	persistentPeers := make([]string, nValidators+nNonValidators)
	for i := 0; i < nValidators+nNonValidators; i++ {
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("%s%d", nodeDirPrefix, i))
		config.SetRoot(nodeDir)
		nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
		if err != nil {
			return err
		}
		persistentPeers[i] = p2p.IDAddressString(nodeKey.ID(), fmt.Sprintf("%s:%d", hostnameOrIP(i), p2pPort))
	}
	persistentPeersList := strings.Join(persistentPeers, ",")

	for i := 0; i < nValidators+nNonValidators; i++ {
		nodeDir := filepath.Join(outputDir, fmt.Sprintf("%s%d", nodeDirPrefix, i))
		config.SetRoot(nodeDir)
		config.P2P.PersistentPeers = persistentPeersList
		config.P2P.AddrBookStrict = false

		// overwrite default config
		cfg.WriteConfigFile(filepath.Join(nodeDir, "config", "config.toml"), config)
	}

	return nil
}
