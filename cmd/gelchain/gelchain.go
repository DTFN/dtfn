package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"

	"github.com/tendermint/tendermint/abci/server"

	cmn "github.com/tendermint/tendermint/libs/common"

	abciApp "github.com/green-element-chain/gelchain/app"
	emtUtils "github.com/green-element-chain/gelchain/cmd/utils"
	"github.com/green-element-chain/gelchain/ethereum"
	"github.com/green-element-chain/gelchain/types"
	"github.com/green-element-chain/gelchain/utils"
	tmcfg "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/mempool"
	tmNode "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmState "github.com/tendermint/tendermint/state"
	"math/big"
)

func ethermintCmd(ctx *cli.Context) error {
	// Step 1: Setup the go-ethereum node and start it
	node := emtUtils.MakeFullNode(ctx)
	startNode(ctx, node)

	// Setup the ABCI server and start it
	addr := ctx.GlobalString(emtUtils.ABCIAddrFlag.Name)
	abci := ctx.GlobalString(emtUtils.ABCIProtocolFlag.Name)

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
	ethApp, err := abciApp.NewEthermintApplication(backend, rpcClient, types.NewStrategy())
	strategy := ethApp.GetStrategy()
	strategy.BlsSelectStrategy = ctx.GlobalBool(emtUtils.TmBlsSelectStrategy.Name)
	priceBarrier := ctx.GlobalInt64(emtUtils.TxpoolPriceLimit.Name)
	if priceBarrier > 0 {
		strategy.PriceBarrier.SetInt64(priceBarrier)
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ethApp.StartHttpServer()
	ethLogger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)).With("module", "gelchain")
	configLoggerLevel(ctx, &ethLogger)
	ethApp.SetLogger(ethLogger)

	tmConfig := loadTMConfig(ctx)

	hasPersistData := ethApp.InitPersistData()
	if !hasPersistData {
		ethGenesisJson := ethermintGenesisPath(ctx)
		genesis := utils.ReadGenesis(ethGenesisJson)
		totalBalanceInital := big.NewInt(0)
		for key, _ := range genesis.Alloc {
			totalBalanceInital.Add(totalBalanceInital, genesis.Alloc[key].Balance)
		}
		strategy.CurrEpochValData.TotalBalance = totalBalanceInital

		ethAccounts, err := types.GetInitialEthAccountFromFile(tmConfig.InitialEthAccountFile())
		if err != nil {
			panic("Sorry but you don't have initial account file")
		}

		genDocFile := tmConfig.GenesisFile()
		genDoc, err := tmState.MakeGenesisDocFromFile(genDocFile)
		validators := genDoc.Validators
		amlist := &types.AccountMap{
			MapList: make(map[string]*types.AccountMapItem),
		}
		log.Info(fmt.Sprintf("get Initial accountMap len %v. genDoc.Validators len %v",
			len(ethAccounts.EthAccounts), len(validators)))
		for i := 0; i < len(validators); i++ {
			tmAddress := validators[i].PubKey.Address().String()
			blsKey := validators[i].BlsPubKey
			blsKeyJsonStr, _ := json.Marshal(blsKey)
			/*		accountBalance := big.NewInt(1)
					accountBalance.Div(totalBalanceInital, big.NewInt(100))*/
			if i == len(ethAccounts.EthAccounts) {
				break
			}
			amlist.MapList[tmAddress] = &types.AccountMapItem{
				common.HexToAddress(ethAccounts.EthAccounts[i]),
				common.HexToAddress(ethAccounts.EthBeneficiarys[i]), //10个eth账户中的第i个。
				string(blsKeyJsonStr),
			}
		}

		strategy.SetInitialAccountMap(amlist)
		log.Info(fmt.Sprintf("SetInitialAccountMap %v", amlist))
	}
	if strategy.CurrEpochValData.TotalBalance.Int64() == 0 {
		panic("strategy.CurrEpochValData.TotalBalance==0")
	}
	selectCount := ctx.GlobalInt(emtUtils.SelectCount.Name)
	fmt.Println("selectCount", selectCount)
	strategy.CurrEpochValData.SelectCount = selectCount
	selectBlockNumber := ctx.GlobalInt64(emtUtils.SelectBlockNumber.Name)
	fmt.Println("selectBlockNumber", selectBlockNumber)
	selectStrategy := ctx.GlobalBool(emtUtils.SelectStrategy.Name)
	fmt.Println("selectStrategy", selectStrategy)

	// Step 2: If we can invoke `tendermint node`, let's do so
	// in order to make gelchain as self contained as possible.
	// See Issue https://github.com/tendermint/ethermint/issues/244
	canInvokeTendermintNode := canInvokeTendermint(ctx)
	if canInvokeTendermintNode {
		tmConfig := loadTMConfig(ctx)
		clientCreator := proxy.NewLocalClientCreator(ethApp)
		tmLogger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)).With("module", "tendermint")
		configLoggerLevel(ctx, &tmLogger)

		// Generate node PrivKey
		nodeKey, err := p2p.LoadOrGenNodeKey(tmConfig.NodeKeyFile())
		if err != nil {
			return err
		}

		// Convert old PrivValidator if it exists.
		oldPrivVal := tmConfig.OldPrivValidatorFile()
		newPrivValKey := tmConfig.PrivValidatorKeyFile()
		newPrivValState := tmConfig.PrivValidatorStateFile()
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
		n, err := tmNode.NewNode(tmConfig,
			privval.LoadOrGenFilePV(newPrivValKey, newPrivValState),
			nodeKey,
			clientCreator,
			tmNode.DefaultGenesisDocProviderFunc(tmConfig),
			tmNode.DefaultDBProvider,
			tmNode.DefaultMetricsProvider(tmConfig.Instrumentation),
			tmLogger)
		if err != nil {
			log.Info("tendermint newNode", "error", err)
			return err
		}

		rollbackFlag := ctx.GlobalBool(emtUtils.RollbackFlag.Name)
		rollbackHeight := ctx.GlobalInt(emtUtils.RollbackHeight.Name)
		whetherRollbackEthApp(rollbackFlag, rollbackHeight, backend)

		memPool := n.Mempool()
		backend.SetMemPool(memPool)
		clist_mempool := memPool.(*mempool.CListMempool)
		clist_mempool.SetRecheckFailCallback(backend.Ethereum().TxPool().RemoveTxs)

		err = n.Start()
		if err != nil {
			log.Error("server with tendermint start", "error", err)
			return err
		}
		// Stop upon receiving SIGTERM or CTRL-C.
		cmn.TrapSignal(tmLogger, func() {
			if n.IsRunning() {
				n.Stop()
			}
		})

		/*	h := new(memsizeui.Handler)
			s := &http.Server{Addr: "0.0.0.0:7070", Handler: h}
			txs := clist_mempool.Txs()
			sMap := clist_mempool.TxsMap()
			state, _ := backend.Es().State()
			work:=backend.Es().WorkState()
			txPool := backend.Ethereum().TxPool()
			h.Add("syncMap", &sMap)
			h.Add("txsList", txs)
			h.Add("esState", state)
			h.Add("workState", &work)
			txPool.DebugMeomory(h)
			go s.ListenAndServe()*/

		// Run forever.
		select {}
		return nil
	} else {
		// Start the app on the ABCI server
		srv, err := server.NewServer(addr, abci, ethApp)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		logger := emtUtils.EthermintLogger().With("module", "abci-server")
		srv.SetLogger(logger)

		if err := srv.Start(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cmn.TrapSignal(logger, func() {
			if srv.IsRunning() {
				srv.Stop()
			}
		})
		// Run forever.
		select {}
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
	defaultTmConfig.Mempool.Size = ctx.GlobalInt(emtUtils.MempoolSize.Name)
	defaultTmConfig.Mempool.Broadcast = ctx.GlobalBool(emtUtils.MempoolBroadcastFlag.Name)
	defaultTmConfig.Mempool.FlowControl = ctx.GlobalBool(emtUtils.FlowControlFlag.Name)
	defaultTmConfig.Mempool.FlowControlThreshold = ctx.GlobalInt(emtUtils.MempoolThreshold.Name)
	defaultTmConfig.Mempool.FlowControlHeightThreshold = ctx.GlobalInt(emtUtils.MempoolHeightThreshold.Name)
	defaultTmConfig.Mempool.FlowControlMaxSleepTime = time.Duration(ctx.GlobalInt(emtUtils.FlowControlMaxSleepTime.Name)) * time.Second
	defaultTmConfig.P2P.RootDir = tmHome
	defaultTmConfig.RPC.RootDir = tmHome
	defaultTmConfig.Consensus.RootDir = tmHome
	defaultTmConfig.Consensus.CreateEmptyBlocks = ctx.GlobalBool(emtUtils.TmConsEmptyBlock.Name)
	defaultTmConfig.Consensus.CreateEmptyBlocksInterval = time.Duration(ctx.GlobalInt(emtUtils.TmConsEBlockInteval.Name)) * time.Second
	defaultTmConfig.Consensus.NeedProofBlock = ctx.GlobalBool(emtUtils.TmConsNeedProofBlock.Name)
	defaultTmConfig.Consensus.TimeoutPropose = time.Duration(ctx.GlobalInt(emtUtils.TmConsProposeTimeout.Name)) * time.Second

	defaultTmConfig.RollbackHeight = ctx.GlobalInt64(emtUtils.RollbackHeight.Name)
	defaultTmConfig.RollbackFlag = ctx.GlobalBool(emtUtils.RollbackFlag.Name)

	defaultTmConfig.Instrumentation = DefaultInstrumentationConfig

	defaultTmConfig.FastSync = ctx.GlobalBool(emtUtils.FastSync.Name)
	defaultTmConfig.BaseConfig.InitialEthAccount = ctx.GlobalString(emtUtils.TmInitialEthAccount.Name)
	defaultTmConfig.PrivValidatorListenAddr = ctx.GlobalString(emtUtils.PrivValidatorListenAddr.Name)
	defaultTmConfig.PrivValidatorKey = ctx.GlobalString(emtUtils.PrivValidator.Name)
	defaultTmConfig.P2P.AddrBook = ctx.GlobalString(emtUtils.AddrBook.Name)
	defaultTmConfig.P2P.AddrBookStrict = ctx.GlobalBool(emtUtils.RoutabilityStrict.Name)
	defaultTmConfig.P2P.PersistentPeers = ctx.GlobalString(emtUtils.PersistentPeers.Name)
	defaultTmConfig.P2P.PrivatePeerIDs = ctx.GlobalString(emtUtils.PrivatePeerIDs.Name)
	defaultTmConfig.P2P.ListenAddress = ctx.GlobalString(emtUtils.TendermintP2PListenAddress.Name)
	defaultTmConfig.P2P.ExternalAddress = ctx.GlobalString(emtUtils.TendermintP2PExternalAddress.Name)
	defaultTmConfig.P2P.MaxNumInboundPeers = ctx.GlobalInt(emtUtils.MaxInPeers.Name)
	defaultTmConfig.P2P.MaxNumOutboundPeers = ctx.GlobalInt(emtUtils.MaxInPeers.Name)

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

//delete history block and rollback state here
//and should put it before the rollback of tendermint
func whetherRollbackEthApp(rollbackFlag bool, rollbackHeight int, appBackend *ethereum.Backend) {
	if rollbackFlag {
		fmt.Println("you are rollbacking")
		appBackend.Ethereum().BlockChain().RewindTo(uint64(rollbackHeight))
		fmt.Println(appBackend.Ethereum().BlockChain().CurrentBlock().NumberU64())
		os.Exit(1)
	} else {
		fmt.Println(appBackend.GasLimit())
		fmt.Println("You are not rollbacking")
	}
}
