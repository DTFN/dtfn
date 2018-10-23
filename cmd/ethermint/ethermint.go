package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/ethermint/utils"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/tendermint/tendermint/abci/server"

	cmn "github.com/tendermint/tendermint/libs/common"

	abciApp "github.com/tendermint/ethermint/app"
	emtUtils "github.com/tendermint/ethermint/cmd/utils"
	"github.com/tendermint/ethermint/ethereum"
	"github.com/tendermint/ethermint/types"
	tmcfg "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmState "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
)

func ethermintCmd(ctx *cli.Context) error {
	// Step 1: Setup the go-ethereum node and start it
	node := emtUtils.MakeFullNode(ctx)
	startNode(ctx, node)

	// Setup the ABCI server and start it
	addr := ctx.GlobalString(emtUtils.ABCIAddrFlag.Name)
	abci := ctx.GlobalString(emtUtils.ABCIProtocolFlag.Name)

	ethGenesisJson := ethermintGenesisPath(ctx)
	genesis := utils.ReadGenesis(ethGenesisJson)
	totalBalanceInital := big.NewInt(0)
	for key, _ := range genesis.Alloc {
		totalBalanceInital.Add(totalBalanceInital, genesis.Alloc[key].Balance)
	}
	// Fetch the registered service of this type
	var backend *ethereum.Backend
	if err := node.Service(&backend); err != nil {
		ethUtils.Fatalf("ethereum backend service not running: %v", err)
	}

	// In-proc RPC connection so ABCI.Query can be forwarded over the ethereum rpc
	rpcClient, err := node.Attach()
	if err != nil {
		ethUtils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}

	// Create the ABCI app
	ethApp, err := abciApp.NewEthermintApplication(backend, rpcClient, types.NewStrategy(totalBalanceInital))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ethApp.StartHttpServer()
	ethLogger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)).With("module", "ethermint")
	configLoggerLevel(ctx, &ethLogger)
	ethApp.SetLogger(ethLogger)

	amlist, err := tmTypes.AccountMapFromFile(loadTMConfig(ctx).AddressMapFile())
	if err != nil {
		//这里需要自己构造一个新的accountmaplist来用，构造来自tendermint的genesis.json
		tmConfig := loadTMConfig(ctx)
		genDocFile := tmConfig.GenesisFile()

		genDoc, err := tmState.MakeGenesisDocFromFile(genDocFile)
		if err != nil {
			fmt.Println(err)
		}
		validators := genDoc.Validators
		var tmAddress []string
		amlist = &tmTypes.AccountMapList{
			MapList: make(map[string]*tmTypes.AccountMap),
		}
		for i := 0; i < len(validators); i++ {
			tmAddress = append(tmAddress, strings.ToLower(hex.EncodeToString(validators[i].PubKey.Address())))
			accountBalance := big.NewInt(1)
			accountBalance.Div(totalBalanceInital, big.NewInt(100))
			switch i {
			case 0:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f90142"), //10个eth账户中的第一个//
					"1",//
				}
			case 1:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0x8423328b8016fbe31938a461b5647de696bdbf71"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0x8423328b8016fbe31938a461b5647de696bdbf72"), //10个eth账户中的第一个//
					"2",// 。
				}
			case 2:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0x4eba28c09155a61503b2be9cbd3dacf8b84dcfb8"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0x4eba28c09155a61503b2be9cbd3dacf8b84dcfb2"), //10个eth账户中的第一个。
					"3",
				}
			case 3:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0xfc6e050a795ca66139262ddc36bbf8b11ab1911e"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0xfc6e050a795ca66139262ddc36bbf8b11ab19112"), //10个eth账户中的第一个
					"4",// 。
				}
			case 4:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0x99c80ff44e34a462da6cb3a96295106f11b3467a"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0x99c80ff44e34a462da6cb3a96295106f11b34672"), //10个eth账户中的第一个
					"5",// 。
				}
			case 5:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0xe530df4446e4d2885d0564c9bce3cbc478c231b5"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0xe530df4446e4d2885d0564c9bce3cbc478c231b2"), //10个eth账户中的第一个
					"6",// 。
				}
			case 6:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0xade651aad6507678751c1c1e5e32dbd9dc97fa4e"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0xade651aad6507678751c1c1e5e32dbd9dc97fa42"), //10个eth账户中的第一个
					"7",// 。
				}
			case 7:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0x1ae4d63ea5ad162e6fcb1ff94433e9fa8b400464"),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0x1ae4d63ea5ad162e6fcb1ff94433e9fa8b400462"), //10个eth账户中的第一个
					"8",// 。
				}
			default:
				amlist.MapList[tmAddress[i]] = &tmTypes.AccountMap{
					common.HexToAddress("0x0000000000000000000000000000000000000"+strconv.Itoa(100+i)),
					accountBalance,
					big.NewInt(0),
					common.HexToAddress("0x0000000000000000000000000000000000001"+strconv.Itoa(100+i)), //10个eth账户中的第一个
					strconv.Itoa(i),// 。
				}
			}
		}
	}
	ethApp.GetStrategy().SetAccountMapList(amlist)

	// Step 2: If we can invoke `tendermint node`, let's do so
	// in order to make ethermint as self contained as possible.
	// See Issue https://github.com/tendermint/ethermint/issues/244
	canInvokeTendermintNode := canInvokeTendermint(ctx)
	if canInvokeTendermintNode {
		tmConfig := loadTMConfig(ctx)
		clientCreator := proxy.NewLocalClientCreator(ethApp)
		tmLogger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)).With("module", "tendermint")
		configLoggerLevel(ctx, &tmLogger)

		n, err := tmNode.NewNode(tmConfig,
			privval.LoadOrGenFilePV(tmConfig.PrivValidatorFile()),
			clientCreator,
			tmNode.DefaultGenesisDocProviderFunc(tmConfig),
			tmNode.DefaultDBProvider,
			tmNode.DefaultMetricsProvider(tmConfig.Instrumentation),
			tmLogger)
		if err != nil {
			log.Info("tendermint newNode", "error", err)
			return err
		}

		backend.SetMemPool(n.MempoolReactor().Mempool)
		n.MempoolReactor().Mempool.SetRecheckFailCallback(backend.Ethereum().TxPool().RemoveTxs)

		err = n.Start()
		if err != nil {
			log.Error("server with tendermint start", "error", err)
			return err
		}
		// Trap signal, run forever.
		n.RunForever()
		return nil
	} else {
		// Start the app on the ABCI server
		srv, err := server.NewServer(addr, abci, ethApp)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		srv.SetLogger(emtUtils.EthermintLogger().With("module", "abci-server"))

		if err := srv.Start(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cmn.TrapSignal(func() {
			srv.Stop()
		})
	}

	return nil
}

//加载tendermint相关的配置
func loadTMConfig(ctx *cli.Context) *tmcfg.Config {
	tmHome := tendermintHomeFromEthermint(ctx)
	baseConfig := tmcfg.DefaultBaseConfig()
	baseConfig.RootDir = tmHome

	DefaultInstrumentationConfig := tmcfg.DefaultInstrumentationConfig()

	defaultTmConfig := tmcfg.DefaultConfig()
	defaultTmConfig.BaseConfig = baseConfig
	defaultTmConfig.Mempool.RootDir = tmHome
	defaultTmConfig.Mempool.Recheck = true //fix nonce bug
	defaultTmConfig.P2P.RootDir = tmHome
	defaultTmConfig.RPC.RootDir = tmHome
	defaultTmConfig.Consensus.RootDir = tmHome
	defaultTmConfig.Consensus.CreateEmptyBlocks = ctx.GlobalBool(emtUtils.TmConsEmptyBlock.Name)
	defaultTmConfig.Consensus.CreateEmptyBlocksInterval = ctx.GlobalInt(emtUtils.TmConsEBlockInteval.Name)
	defaultTmConfig.Consensus.NeedProofBlock = ctx.GlobalBool(emtUtils.TmConsNeedProofBlock.Name)

	fmt.Println("wenbin test empty block")
	fmt.Println(defaultTmConfig.Consensus.CreateEmptyBlocks)
	fmt.Println(defaultTmConfig.Consensus.CreateEmptyBlocksInterval)
	fmt.Println("wenbin test empty block")

	defaultTmConfig.Instrumentation = DefaultInstrumentationConfig

	defaultTmConfig.FastSync = ctx.GlobalBool(emtUtils.FastSync.Name)
	defaultTmConfig.PrivValidatorListenAddr = ctx.GlobalString(emtUtils.PrivValidatorListenAddr.Name)
	defaultTmConfig.PrivValidator = ctx.GlobalString(emtUtils.PrivValidator.Name)
	defaultTmConfig.P2P.AddrBook = ctx.GlobalString(emtUtils.AddrBook.Name)
	defaultTmConfig.P2P.AddrBookStrict = ctx.GlobalBool(emtUtils.RoutabilityStrict.Name)
	defaultTmConfig.P2P.PersistentPeers = ctx.GlobalString(emtUtils.PersistentPeers.Name)
	defaultTmConfig.P2P.PrivatePeerIDs = ctx.GlobalString(emtUtils.PrivatePeerIDs.Name)
	defaultTmConfig.P2P.ListenAddress = ctx.GlobalString(emtUtils.TendermintP2PListenAddress.Name)
	defaultTmConfig.P2P.ExternalAddress = ctx.GlobalString(emtUtils.TendermintP2PExternalAddress.Name)

	return defaultTmConfig
}

func configLoggerLevel(ctx *cli.Context, logger *tmlog.Logger) {
	switch ctx.GlobalString(emtUtils.LogLevelFlag.Name) {
	case "error":
		*logger = tmlog.NewFilter(*logger, tmlog.AllowError())
	case "info":
		*logger = tmlog.NewFilter(*logger, tmlog.AllowInfo())
	default:
		*logger = tmlog.NewFilter(*logger, tmlog.AllowAll())
	}
}

// nolint
// startNode copies the logic from go-ethereum
func startNode(ctx *cli.Context, stack *ethereum.Node) {
	emtUtils.StartNode(stack)

	// Unlock any account specifically requested
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	passwords := ethUtils.MakePasswordList(ctx)
	unlocks := strings.Split(ctx.GlobalString(ethUtils.UnlockedAccountFlag.Name), ",")
	for i, account := range unlocks {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, ks, trimmed, i, passwords)
		}
	}
	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	go func() {
		// Create an chain state reader for self-derivation
		rpcClient, err := stack.Attach()
		if err != nil {
			ethUtils.Fatalf("Failed to attach to self: %v", err)
		}
		stateReader := ethclient.NewClient(rpcClient)

		// Open and self derive any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			} else {
				wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				if event.Wallet.URL().Scheme == "ledger" {
					event.Wallet.SelfDerive(accounts.DefaultLedgerBaseDerivationPath, stateReader)
				} else {
					event.Wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
				}

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()
}

// tries unlocking the specified account a few times.
// nolint: unparam
func unlockAccount(ctx *cli.Context, ks *keystore.KeyStore, address string, i int,
	passwords []string) (accounts.Account, string) {

	account, err := ethUtils.MakeAddress(ks, address)
	if err != nil {
		ethUtils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	ethUtils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

// getPassPhrase retrieves the passwor associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
// nolint: unparam
func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		ethUtils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			ethUtils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			ethUtils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError,
	auth string) accounts.Account {

	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your passphrase against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		ethUtils.Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your passphrase unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}
