package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/DTFN/dtfn/version"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txfilter"
	"github.com/ethereum/go-ethereum/core/types"
	ethereumCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	//_ "net/http/pprof"
)

// format of query data
type jsonRequest struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id,omitempty"`
	Params []interface{}   `json:"params,omitempty"`
}

// rlp decode an etherum transaction
func decodeTx(txBytes []byte) (*types.Transaction, error) {
	tx := new(types.Transaction)
	rlpStream := rlp.NewStream(bytes.NewBuffer(txBytes), 0)
	if err := tx.DecodeRLP(rlpStream); err != nil {
		return nil, err
	}
	return tx, nil
}

//-------------------------------------------------------
// convenience methods for validators

// Receiver returns the receiving address based on the selected strategy
// #unstable
func (app *EthermintApplication) Receiver() common.Address {
	if app.strategy != nil {
		return app.strategy.Receiver()
	}
	return common.Address{}
}

func (app *EthermintApplication) StartHttpServer() {
	go app.httpServer.HttpServer.ListenAndServe()
	//go http.ListenAndServe("0.0.0.0:6060", nil)
}

// GetUpdatedValidators returns an updated validator set from the strategy
// #unstable
func (app *EthermintApplication) GetUpdatedValidators(height int64, seed []byte) abciTypes.ResponseEndBlock {
	if app.strategy != nil {
		return app.strategy.GetUpdatedValidators(height, seed)
	}
	return abciTypes.ResponseEndBlock{AppVersion: app.strategy.HFExpectedData.BlockVersion}
}

func (app *EthermintApplication) SetPosTableThreshold() {
	if app.strategy.CurrEpochValData.TotalBalance.Int64() == 0 {
		panic("strategy.CurrEpochValData.TotalBalance==0")
	}
	thresholdUnit := big.NewInt(txfilter.ThresholdUnit)
	threshold := big.NewInt(0)
	threshold.Div(app.strategy.CurrEpochValData.TotalBalance, thresholdUnit)
	app.strategy.NextEpochValData.PosTable.SetThreshold(threshold)
}

func (app *EthermintApplication) InitPersistData() bool {
	app.logger.Info("Init Persist Data")

	core.EvmErrHardForkHeight = version.EvmErrHardForkHeight

	txfilter.Bigguy = common.HexToAddress(version.Bigguy)

	wsState, _ := app.backend.Es().State()

	currEpochDataAddress := txfilter.SendToLock

	trie := wsState.StorageTrie(currEpochDataAddress)

	app.logger.Info("Read CurrEpochValData")
	currBytes := wsState.GetCode(currEpochDataAddress)

	app.logger.Info("Read currentHeightData")
	key := []byte("CurrentHeightData")
	keyHash := common.BytesToHash(key)
	valueHash := wsState.GetState(currEpochDataAddress, keyHash)
	if bytes.Equal(valueHash.Bytes(), common.Hash{}.Bytes()) {
		app.logger.Info("no pre CurrentHeightData")
		app.strategy.NextEpochValData.PosTable = txfilter.CreatePosTable()
		app.strategy.AuthTable = txfilter.CreateAuthTable()
		return false
	} else {
		currentHeightData, err := trie.TryGet(key)
		if err != nil {
			panic(fmt.Sprintf("resolve currentHeightData err %v", err))
		}
		if len(currentHeightData) == 0 {
			// no predata existed
			panic("len(currentHeightData) == 0")
		} else {
			app.logger.Info("currentHeightData Not nil")
			err := json.Unmarshal(currentHeightData, &app.strategy.CurrentHeightValData)
			if err != nil {
				panic(fmt.Sprintf("initialize CurrentHeightValData.Validators error %v", err))
			}
		}
	}
	app.strategy.HFExpectedData.Height = app.strategy.CurrentHeightValData.Height
	if app.strategy.HFExpectedData.IsHarfForkPassed {
		for i := len(version.HeightArray) - 1; i >= 0; i-- {
			if app.strategy.HFExpectedData.Height >= version.HeightArray[i] {
				app.strategy.HFExpectedData.BlockVersion = uint64(version.VersionArray[i])
				break
			}
		}
	}
	txfilter.AppVersion = app.strategy.HFExpectedData.BlockVersion
	if txfilter.AppVersion <= 4 {
		txfilter.PPChainAdmin = common.HexToAddress(version.PPChainAdmin)
	} else {
		txfilter.PPChainAdmin = common.HexToAddress(version.PPChainPrivateAdmin)
	}

	if len(currBytes) == 0 {
		// no predata existed
		panic("no pre CurrEpochValData")
	} else {
		app.logger.Info("CurrEpochValData Not nil")
		err := json.Unmarshal(currBytes, &app.strategy.CurrEpochValData)
		if err != nil {
			panic(fmt.Sprintf("initialize CurrEpochValData error %v", err))
		} else {
			app.strategy.CurrEpochValData.PosTable.InitStruct()
			app.strategy.CurrEpochValData.PosTable.ExportSortedSigners()
			txfilter.CurrentPosTable = app.strategy.CurrEpochValData.PosTable
		}
	}

	app.logger.Info("Read PosTable")
	app.strategy.NextEpochValData.PosTable = wsState.InitPosTable()
	app.strategy.NextEpochValData.PosTable.InitStruct()

	app.logger.Info("Read AuthTable")
	app.strategy.AuthTable = wsState.InitAuthTable()

	return true
}

func (app *EthermintApplication) SetPersistenceData() {
	wsState, _ := app.getCurrentState()
	height := app.strategy.CurrentHeightValData.Height
	app.logger.Info(fmt.Sprintf("set persist data in height %v", height))
	nextEpochDataAddress := txfilter.SendToUnlock
	currEpochDataAddress := txfilter.SendToLock

	// we didn't need reset the slots of postable because it it right now.
	//if height == version.HeightArray[2] {
	//	//height??
	//	for index, value := range app.strategy.NextEpochValData.PosTable.PosItemMap {
	//		fmt.Println(index)
	//		(*value).Slots = 10
	//		app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock = true
	//		fmt.Println((*value).Slots)
	//	}
	//	app.strategy.NextEpochValData.PosTable.TotalSlots = int64(len(app.strategy.NextEpochValData.PosTable.PosItemMap)) * 10
	//	app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock = true
	//}

	if app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock || height%txfilter.EpochBlocks == 0 {
		nextBytes, _ := json.Marshal(app.strategy.NextEpochValData.PosTable)
		wsState.SetCode(nextEpochDataAddress, nextBytes)
		app.logger.Debug(fmt.Sprintf("NextEpochValData.PosTable %v", app.strategy.NextEpochValData.PosTable))
		if app.strategy.HFExpectedData.BlockVersion >= 4 {
			nextBytes, _ = json.Marshal(app.strategy.AuthTable)
			wsState.SetCode(txfilter.SendToAuth, nextBytes)
			if app.strategy.HFExpectedData.BlockVersion >= 5 {
				trie := wsState.GetOrNewStateObject(txfilter.SendToAuth).GetTrie(wsState.Database())
				key := []byte("ExtendAuthTable")
				keyHash := common.BytesToHash(key)
				//persist every height
				valBytes, _ := json.Marshal(app.strategy.AuthTable.ExtendAuthTable)
				trie.TryUpdate(key, valBytes)
				valueHash := ethereumCrypto.Keccak256Hash(valBytes)
				wsState.SetState(txfilter.SendToAuth, keyHash, valueHash)
			}

			app.logger.Debug(fmt.Sprintf("AuthTable %v", app.strategy.AuthTable))
		}
	}

	if height%txfilter.EpochBlocks == 0 {
		currBytes, _ := json.Marshal(app.strategy.CurrEpochValData)
		wsState.SetCode(currEpochDataAddress, currBytes)
	}

	trie := wsState.GetOrNewStateObject(currEpochDataAddress).GetTrie(wsState.Database())
	key := []byte("CurrentHeightData")
	keyHash := common.BytesToHash(key)
	//persist every height
	valBytes, _ := json.Marshal(app.strategy.CurrentHeightValData)
	trie.TryUpdate(key, valBytes)
	valueHash := ethereumCrypto.Keccak256Hash(valBytes)
	wsState.SetState(currEpochDataAddress, keyHash, valueHash)
	app.logger.Debug(fmt.Sprintf("CurrentHeightValData %v", app.strategy.CurrentHeightValData))
}
