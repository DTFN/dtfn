package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txfilter"
	"github.com/ethereum/go-ethereum/core/types"
	ethereumCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	ethmintTypes "github.com/green-element-chain/gelchain/types"
	"github.com/green-element-chain/gelchain/version"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tmTypes "github.com/tendermint/tendermint/types"
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

// SetValidators sets new validators on the strategy
// #unstable
func (app *EthermintApplication) SetValidators(validators []abciTypes.ValidatorUpdate) {
	if app.strategy != nil {
		app.strategy.SetValidators(validators)
	}
}

func (app *EthermintApplication) StartHttpServer() {
	go app.httpServer.HttpServer.ListenAndServe()
	//go http.ListenAndServe("0.0.0.0:6060", nil)
}

// GetUpdatedValidators returns an updated validator set from the strategy
// #unstable
func (app *EthermintApplication) GetUpdatedValidators(height int64, seed []byte) abciTypes.ResponseEndBlock {
	if app.strategy != nil {
		if height%txfilter.EpochBlocks != 0 {
			if seed != nil {
				//seed 存在的时，优先seed
				return app.enterSelectValidators(seed, -1)
			} else {
				//seed 不存在，选取height
				return app.enterSelectValidators(nil, height)
			}
		} else {
			return app.blsValidators(height)
		}
	}
	return abciTypes.ResponseEndBlock{AppVersion: app.strategy.HFExpectedData.BlockVersion}
}

// CollectTx invokes CollectTx on the strategy
// #unstable
func (app *EthermintApplication) CollectTx(tx *types.Transaction) {
	if app.strategy != nil {
		app.strategy.CollectTx(tx)
	}
}

func (app *EthermintApplication) enterSelectValidators(seed []byte, height int64) abciTypes.ResponseEndBlock {
	/*if app.strategy.BlsSelectStrategy {
	} else {
	}*/
	validatorsSlice := []abciTypes.ValidatorUpdate{}

	selectCount := app.strategy.CurrEpochValData.SelectCount //currently fixed
	poolLen := len(app.strategy.CurrEpochValData.PosTable.PosItemMap)
	if poolLen < 7 {
		app.GetLogger().Info(fmt.Sprintf("PosTable.PosItemMap len < 7, current len %v", poolLen))
	}
	if selectCount == 0 { //0 means return full set each height
		selectCount = poolLen
	}

	// we use map to remember which validators selected has put into validatorSlice
	selectedValidators := make(map[string]int)

	if app.strategy.HFExpectedData.BlockVersion >= 2 {
		for i := 0; len(validatorsSlice) != selectCount; i++ {
			var tmPubKey crypto.PubKey
			var validator ethmintTypes.Validator
			var signer common.Address
			var pubKey abciTypes.PubKey
			var posItem txfilter.PosItem
			if height == -1 {
				//height=-1 表示 seed 存在，使用seed
				signer, posItem = app.strategy.CurrEpochValData.PosTable.SelectItemBySeedValue(seed, i)
			} else {
				//seed 不存在，使用height
				startIndex := height
				signer, posItem = app.strategy.CurrEpochValData.PosTable.SelectItemByHeightValue(startIndex + int64(i))
			}
			pubKey = posItem.PubKey
			tmPubKey, _ = tmTypes.PB2TM.PubKey(pubKey)
			tmAddress := tmPubKey.Address().String()
			if index, ok := selectedValidators[tmAddress]; ok {
				validatorsSlice[index].Power++
			} else {
				validatorUpdate := abciTypes.ValidatorUpdate{
					PubKey: pubKey,
					Power:  1000,
				}
				validator = ethmintTypes.Validator{
					validatorUpdate,
					signer,
				}
				//Remember tmPubKey.Address 's index in the currentValidators Array
				selectedValidators[tmAddress] = len(validatorsSlice)
				validatorsSlice = append(validatorsSlice, validatorUpdate)
				app.strategy.CurrentHeightValData.Validators[tmAddress] = validator
			}
		}
	} else {
		//select validators from posTable
		for i := 0; i < selectCount; i++ {
			var tmPubKey crypto.PubKey
			var validator ethmintTypes.Validator
			var signer common.Address
			var pubKey abciTypes.PubKey
			var posItem txfilter.PosItem
			if height == -1 {
				//height=-1 表示 seed 存在，使用seed
				signer, posItem = app.strategy.CurrEpochValData.PosTable.SelectItemBySeedValue(seed, i)
			} else {
				//seed 不存在，使用height
				startIndex := height
				signer, posItem = app.strategy.CurrEpochValData.PosTable.SelectItemByHeightValue(startIndex + int64(i))
			}
			pubKey = posItem.PubKey
			tmPubKey, _ = tmTypes.PB2TM.PubKey(pubKey)
			tmAddress := tmPubKey.Address().String()
			if index, ok := selectedValidators[tmAddress]; ok {
				validatorsSlice[index].Power++
			} else {
				validatorUpdate := abciTypes.ValidatorUpdate{
					PubKey: pubKey,
					Power:  1000,
				}
				validator = ethmintTypes.Validator{
					validatorUpdate,
					signer,
				}
				//Remember tmPubKey.Address 's index in the currentValidators Array
				selectedValidators[tmAddress] = len(validatorsSlice)
				validatorsSlice = append(validatorsSlice, validatorUpdate)
				app.strategy.CurrentHeightValData.Validators[tmAddress] = validator
			}
		}
	}

	//append the validators which will be deleted
	for address, v := range app.strategy.CurrentHeightValData.Validators {
		//tmPubKey, _ := tmTypes.PB2TM.PubKey(v.PubKey)
		index, selected := selectedValidators[address]
		if selected {
			v.Power = validatorsSlice[index].Power
		} else {
			validatorsSlice = append(validatorsSlice, abciTypes.ValidatorUpdate{
				PubKey: v.PubKey,
				Power:  0,
			})
			delete(app.strategy.CurrentHeightValData.Validators, address)
		}
	}

	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
}

func (app *EthermintApplication) blsValidators(height int64) abciTypes.ResponseEndBlock {
	blsPubkeySlice := []string{}
	validatorsSlice := []abciTypes.ValidatorUpdate{}
	topKSigners := app.strategy.CurrEpochValData.PosTable.TopKSigners(100)
	currentValidators := map[string]ethmintTypes.Validator{}

	for _, signer := range topKSigners {
		posItem := app.strategy.CurrEpochValData.PosTable.PosItemMap[signer]
		tmAddress := posItem.TmAddress
		updateValidator := abciTypes.ValidatorUpdate{
			PubKey: posItem.PubKey,
			Power:  posItem.Slots,
		}
		emtValidator := ethmintTypes.Validator{updateValidator, signer}
		currentValidators[tmAddress] = emtValidator
		validatorsSlice = append(validatorsSlice, updateValidator)
		blsPubkeySlice = append(blsPubkeySlice, posItem.BlsKeyString)
	}

	for tmAddress, v := range app.strategy.CurrentHeightValData.Validators {
		_, ok := currentValidators[tmAddress]
		if !ok {
			validatorsSlice = append(validatorsSlice,
				abciTypes.ValidatorUpdate{
					PubKey: v.PubKey,
					Power:  int64(0),
				})
		}
	}
	app.strategy.CurrentHeightValData.Validators = currentValidators

	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, BlsKeyString: blsPubkeySlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
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
	// marshal map to jsonBytes,is it sorted?
	wsState, _ := app.backend.Es().State()

	//nextEpochDataAddress := common.HexToAddress("0x8888888888888888888888888888888888888888")
	currEpochDataAddress := common.HexToAddress("0x7777777777777777777777777777777777777777")

	trie := wsState.StorageTrie(currEpochDataAddress)

	/*	app.logger.Info("Read NextEpochValData")
		nextBytes := wsState.GetCode(nextEpochDataAddress)*/

	app.logger.Info("Read CurrEpochValData")
	currBytes := wsState.GetCode(currEpochDataAddress)

	app.logger.Info("Read currentHeightData")
	key := []byte("CurrentHeightData")
	keyHash := common.BytesToHash(key)
	valueHash := wsState.GetState(currEpochDataAddress, keyHash)
	if bytes.Equal(valueHash.Bytes(), common.Hash{}.Bytes()) {
		app.logger.Info("no pre CurrentHeightData")
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

	/*	if len(nextBytes) == 0 {
			// no predata existed
			app.logger.Info("no pre NextEpochValData")
		} else {
			app.logger.Info("NextEpochValData Not nil")
			err := json.Unmarshal(nextBytes, &app.strategy.NextEpochValData)
			if err != nil {
				panic(fmt.Sprintf("initialize NextEpochValData error %v", err))
			} else {
				initFlag = true
			}
		}*/

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
		}
	}

	app.logger.Info("Read PosTable")
	app.strategy.NextEpochValData.PosTable = wsState.InitPosTable()
	if app.strategy.NextEpochValData.PosTable == nil {
		panic("no pre NextEpochValData.PosTable")
	} else {
		app.strategy.NextEpochValData.PosTable.InitStruct()
	}

	return true
}

func (app *EthermintApplication) SetPersistenceData() {
	wsState, _ := app.getCurrentState()
	height := app.strategy.CurrentHeightValData.Height
	app.logger.Info(fmt.Sprintf("set persist data in height %v", height))
	nextEpochDataAddress := common.HexToAddress("0x8888888888888888888888888888888888888888")
	currEpochDataAddress := common.HexToAddress("0x7777777777777777777777777777777777777777")

	if height == version.HeightArray[2] {
		for index, value := range app.strategy.NextEpochValData.PosTable.PosItemMap {
			fmt.Println(index)
			(*value).Slots = 10
			fmt.Println((*value).Slots)
		}
		app.strategy.NextEpochValData.PosTable.TotalSlots = int64(len(app.strategy.NextEpochValData.PosTable.PosItemMap)) * 10
		app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock = true
	}

	if app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock || height%txfilter.EpochBlocks == 0 {
		nextBytes, _ := json.Marshal(app.strategy.NextEpochValData.PosTable)
		wsState.SetCode(nextEpochDataAddress, nextBytes)
		app.logger.Info(fmt.Sprintf("NextEpochValData.PosTable %v", app.strategy.NextEpochValData.PosTable))
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

	//save specify postable into the statedb,wenbin add
	specifyHeightDataAddress := common.HexToAddress("0x1111111111111111111111111111111111111111")
	if height == version.HeightArray[2] {
		curBytes, _ := json.Marshal(app.strategy.CurrEpochValData)
		wsState.SetCode(specifyHeightDataAddress, curBytes)
	}

	//extract data from code
	//savedSpeifyHeightData := common.HexToAddress("0x1111111111111111111111111111111111111111")
	//nextBytes := wsState.GetCode(savedSpeifyHeightData)
	//fmt.Println("------------------speicfy height data------------------------")
	//fmt.Println(string(nextBytes))
	//fmt.Println("------------------speicfy height data------------------------")
}
