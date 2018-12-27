package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/blacklist"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	ethmintTypes "github.com/green-element-chain/gelchain/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
	"strconv"
	"strings"
	"fmt"
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
}

func (app *EthermintApplication) UpsertValidatorTx(signer common.Address, currentHeight *big.Int, balance *big.Int,
	beneficiary common.Address, pubKey crypto.PubKey, blsKeyString string) (bool, error) {
	app.GetLogger().Info("You are upsert ValidatorTxing")

	if pubKey == nil || len(blsKeyString) == 0 {
		app.GetLogger().Info("nil validator pubkey or bls pubkey")
		return false, errors.New("nil validator pubkey or bls pubkey")
	}
	if app.strategy != nil {
		// judge whether is a valid addValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		abciPubKey := tmTypes.TM2PB.PubKey(pubKey)

		tmAddress := strings.ToLower(hex.EncodeToString(pubKey.Address()))
		app.GetLogger().Info("blsKeyString: " + blsKeyString)
		app.GetLogger().Info("tmAddress: " + tmAddress)

		signerExisted := false
		blsExisted := false

		existValidator, existFlag := app.strategy.NextEpochValData.NextCandidateValidators[tmAddress]

		if existFlag {
			origSigner := app.strategy.NextEpochValData.NextAccountMap.MapList[tmAddress].Signer
			if origSigner.String() != signer.String() {
				app.GetLogger().Info("validator was voted by another signer")
				return false, errors.New("validator was voted by another signer")
			}
			if app.strategy.NextEpochValData.NextAccountMap.MapList[tmAddress].BlsKeyString != blsKeyString {
				return false, errors.New("bls pubKey has been used by other people")
			}
		}
		//做这件事之前必须确认这个signer，不是MapList中已经存在的。
		//1.signer相同，可能来作恶;  2.signer相同，可能不作恶，因为有相同maplist;  3.signer不相同

		for _, v := range app.strategy.NextEpochValData.NextAccountMap.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				signerExisted = true
				break
			}
		}

		for _, v := range app.strategy.NextEpochValData.NextAccountMap.MapList {
			if v.BlsKeyString == blsKeyString {
				blsExisted = true
				break
			}
		}

		stateDb, _ := app.getCurrentState()

		if !signerExisted && !existFlag && !blsExisted {
			if blacklist.IsLock(stateDb, currentHeight.Int64(), signer) {
				lockHeight := blacklist.LockHeight(stateDb, signer)
				app.GetLogger().Info("signer is locked " + strconv.FormatInt(lockHeight, 10))
				return false, errors.New("signer is locked " + strconv.FormatInt(lockHeight, 10))
			}
			// signer不相同 signer should not be locked
			// If is a valid addValidatorTx,change the data in the strategy
			// Should change the maplist and postable and nextCandidateValidator
			upsertFlag, err := app.strategy.NextEpochValData.NextPosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Info(err.Error())
				return false, err
			}
			app.strategy.NextEpochValData.NextAccountMap.MapList[tmAddress] = &ethmintTypes.AccountMapItem{
				Beneficiary:  beneficiary,
				Signer:       signer,
				BlsKeyString: blsKeyString,
			}
			app.strategy.NextEpochValData.NextCandidateValidators[tmAddress] =
				abciTypes.ValidatorUpdate{
					PubKey: abciPubKey,
					Power:  1,
				}
			app.GetLogger().Info("add Validator Tx success")
			app.strategy.NextEpochValData.ChangedFlagThisBlock = true
			return true, nil
		} else if existFlag && signerExisted && blsExisted {
			if !bytes.Equal(existValidator.PubKey.Data, abciPubKey.Data) {
				return false, errors.New(fmt.Sprintf("pubKey %v doesn't match with existing one %v", existValidator.PubKey, abciPubKey))
			}
			//同singer，同MapList[tmAddress]，是来改动balance的
			upsertFlag, err := app.strategy.NextEpochValData.NextPosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Error(err.Error())
				return false, err
			}
			app.strategy.NextEpochValData.NextAccountMap.MapList[tmAddress].Beneficiary = beneficiary
			app.GetLogger().Info("upsert Validator Tx success")
			app.strategy.NextEpochValData.ChangedFlagThisBlock = true
			return true, nil
		} else {
			//signer,validator key,bls key 不符合一致性条件 来捣乱的
			app.GetLogger().Info("signer,validator key ,bls key should keep accordance")
			return false, errors.New("signer,validator key ,bls key should keep accordance")
		}

	}
	return false, errors.New("upsertFailed for unknown reason")
}

func (app *EthermintApplication) RemoveValidatorTx(signer common.Address) (bool, error) {
	app.GetLogger().Info("You are removeValidatorTx")
	if app.strategy != nil {
		//找到tmAddress，另这个的signer与输入相等
		var tmAddress string
		existFlag := false
		for k, v := range app.strategy.NextEpochValData.NextAccountMap.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				tmAddress = k
				existFlag = true
				break
			}
		}

		if !existFlag {
			return false, errors.New(fmt.Sprintf("signer %v not exist in AccountMap", signer))
		}

		if len(app.strategy.NextEpochValData.NextCandidateValidators) <= 4 {
			app.GetLogger().Info("can not remove validator for consensus safety")
			return false, errors.New("can not remove validator for consensus safety")
		}

		// judge whether is a valid removeValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		//tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		_, ok := app.strategy.NextEpochValData.NextCandidateValidators[tmAddress]

		if !ok {
			panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in NextCandidateValidators", signer))
		}
		removeFlag, err := app.strategy.NextEpochValData.NextPosTable.RemovePosItem(signer)
		if err != nil || !removeFlag {
			panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in posTable", signer))
		}

		delete(app.strategy.NextEpochValData.NextCandidateValidators, tmAddress)
		delete(app.strategy.NextEpochValData.NextAccountMap.MapList, tmAddress)
		//if validator is exist in the currentValidators,it must be removed
		app.GetLogger().Info("remove validatorTx success")
		app.strategy.NextEpochValData.ChangedFlagThisBlock = true
		return true, nil
	}
	return false, errors.New("app strategy nil")
}

func (app *EthermintApplication) SetThreshold(threShold *big.Int) {
	if app.strategy != nil {
		app.strategy.CurrHeightValData.PosTable.SetThreShold(threShold)
	}
}

// GetUpdatedValidators returns an updated validator set from the strategy
// #unstable
func (app *EthermintApplication) GetUpdatedValidators(height int64, seed []byte) abciTypes.ResponseEndBlock {
	if app.strategy != nil {
		if int(height) == 1 {
			return app.enterInitial(height)
		} else if int(height)%200 != 0 {
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

func (app *EthermintApplication) enterInitial(height int64) abciTypes.ResponseEndBlock {
	if len(app.strategy.InitialValidators) == 0 {
		// There is no nextCandidateValidators for initial height
		return abciTypes.ResponseEndBlock{AppVersion: app.strategy.HFExpectedData.BlockVersion}
	} else {
		var validatorsSlice []abciTypes.ValidatorUpdate
		validators := app.strategy.GetUpdatedValidators()

		if len(app.strategy.CurrHeightValData.CurrCandidateValidators) == 0 {
			return abciTypes.ResponseEndBlock{AppVersion: app.strategy.HFExpectedData.BlockVersion}
		}

		maxValidators := 0
		if len(app.strategy.CurrHeightValData.CurrCandidateValidators) < 7 {
			maxValidators = len(app.strategy.CurrHeightValData.CurrCandidateValidators)
		} else {
			maxValidators = 7
		}

		votedValidators := make(map[string]bool)
		votedValidatorsIndex := make(map[string]int)

		//select validators from posTable
		for j := 0; len(validatorsSlice) != maxValidators; j++ {
			_, posItem := app.strategy.CurrHeightValData.PosTable.SelectItemByHeightValue(int(height) + j - 1)
			pubKey := posItem.PubKey
			tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
			validator := abciTypes.ValidatorUpdate{
				PubKey: pubKey,
				Power:  1000,
			}
			if votedValidators[tmPubKey.Address().String()] {
				validatorsSlice[votedValidatorsIndex[tmPubKey.Address().String()]].Power++
			} else {
				//Remember tmPubKey.Address 's index in the currentValidators Array
				votedValidatorsIndex[tmPubKey.Address().String()] = len(validatorsSlice)

				votedValidators[tmPubKey.Address().String()] = true
				validatorsSlice = append(validatorsSlice, validator)
				app.strategy.CurrHeightValData.UpdateValidators = append(app.
					strategy.CurrHeightValData.UpdateValidators, validator)
			}
		}

		//record the currentValidator weight for accumulateReward
		for i := 0; i < maxValidators; i++ {
			app.strategy.CurrHeightValData.UpdateValidatorWeights = append(
				app.strategy.CurrHeightValData.UpdateValidatorWeights,
				validatorsSlice[i].Power-999)
		}

		//append the validators which will be deleted
		for i := 0; i < len(validators); i++ {
			tmPubKey, _ := tmTypes.PB2TM.PubKey(validators[i].PubKey)
			if !votedValidators[tmPubKey.Address().String()] {
				validatorsSlice = append(validatorsSlice,
					abciTypes.ValidatorUpdate{
						//Address : app.strategy.PosTable.SelectItemByRandomValue(int(height)).Address,
						Power:  int64(0),
						PubKey: validators[i].PubKey,
					})
			}
		}
		return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
	}
}

func (app *EthermintApplication) enterSelectValidators(seed []byte, height int64) abciTypes.ResponseEndBlock {

	if app.strategy.BlsSelectStrategy {
	} else {
	}

	var validatorsSlice []abciTypes.ValidatorUpdate
	var valCopy []abciTypes.ValidatorUpdate

	maxValidatorSlice := 0
	if len(app.strategy.CurrHeightValData.CurrCandidateValidators) <= 4 {
		return abciTypes.ResponseEndBlock{AppVersion: app.strategy.HFExpectedData.BlockVersion}
	} else if len(app.strategy.CurrHeightValData.CurrCandidateValidators) < 7 {
		maxValidatorSlice = len(app.strategy.CurrHeightValData.CurrCandidateValidators)
	} else {
		maxValidatorSlice = 7
	}

	for i := 0; i < len(app.strategy.CurrHeightValData.UpdateValidators); i++ {
		valCopy = append(valCopy, app.strategy.CurrHeightValData.UpdateValidators[i])
	}

	app.strategy.CurrHeightValData.UpdateValidators = nil
	app.strategy.CurrHeightValData.UpdateValidatorWeights = nil

	// we use map to remember which validators voted has put into validatorSlice
	votedValidators := make(map[string]bool)
	votedValidatorsIndex := make(map[string]int)

	//select validators from posTable
	for i := 0; len(validatorsSlice) != maxValidatorSlice; i++ {
		var tmPubKey crypto.PubKey
		var validator abciTypes.ValidatorUpdate
		if height == -1 {
			//height=-1 表示 seed 存在，使用seed
			_, posItem := app.strategy.CurrHeightValData.PosTable.SelectItemBySeedValue(seed, i)
			pubKey := posItem.PubKey
			tmPubKey, _ = tmTypes.PB2TM.PubKey(pubKey)
			validator = abciTypes.ValidatorUpdate{
				PubKey: pubKey,
				Power:  1000,
			}
		} else {
			//seed 不存在，使用height
			startIndex := int(height) * 100
			_, posItem := app.strategy.CurrHeightValData.PosTable.SelectItemByHeightValue(startIndex + i - 1)
			pubKey := posItem.PubKey
			tmPubKey, _ = tmTypes.PB2TM.PubKey(pubKey)
			validator = abciTypes.ValidatorUpdate{
				PubKey: pubKey,
				Power:  1000,
			}
		}

		if votedValidators[tmPubKey.Address().String()] {
			validatorsSlice[votedValidatorsIndex[tmPubKey.Address().String()]].Power++
		} else {
			//Remember tmPubKey.Address 's index in the currentValidators Array
			votedValidatorsIndex[tmPubKey.Address().String()] = len(validatorsSlice)

			votedValidators[tmPubKey.Address().String()] = true
			validatorsSlice = append(validatorsSlice, validator)
			app.strategy.CurrHeightValData.UpdateValidators = append(app.
				strategy.CurrHeightValData.UpdateValidators, validator)
		}
	}

	//record the currentValidator weight for accumulateReward
	for i := 0; i < maxValidatorSlice; i++ {
		app.strategy.CurrHeightValData.UpdateValidatorWeights = append(
			app.strategy.CurrHeightValData.UpdateValidatorWeights,
			validatorsSlice[i].Power-999)
	}

	//append the validators which will be deleted
	for i := 0; i < len(valCopy); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(valCopy[i].PubKey)
		if !votedValidators[tmPubKey.Address().String()] {
			validatorsSlice = append(validatorsSlice,
				abciTypes.ValidatorUpdate{
					PubKey: valCopy[i].PubKey,
					Power:  int64(0),
				})
		}
	}
	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
}

func (app *EthermintApplication) blsValidators(height int64) abciTypes.ResponseEndBlock {
	blsPubkeySlice := []string{}
	poolValidators := []abciTypes.ValidatorUpdate{}
	updateValidators := []abciTypes.ValidatorUpdate{}
	topKSignerMap := app.strategy.NextEpochValData.NextPosTable.TopKPosItem(100)

	for _, validator := range app.strategy.NextEpochValData.NextCandidateValidators {

		pubkey, _ := tmTypes.PB2TM.PubKey(validator.PubKey)
		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		signer := app.strategy.NextEpochValData.NextAccountMap.MapList[tmAddress].Signer

		if topKSignerMap[signer] != nil {
			power := big.NewInt(1)
			signBalance := app.strategy.NextEpochValData.NextPosTable.PosItemMap[signer].Balance
			power.Div(signBalance,
				app.strategy.NextEpochValData.NextPosTable.Threshold)
			validator := abciTypes.ValidatorUpdate{
				PubKey: app.strategy.NextEpochValData.NextCandidateValidators[tmAddress].PubKey,
				Power:  power.Int64(),
			}
			poolValidators = append(poolValidators, validator)
			updateValidators = append(updateValidators, validator)
			blsPubkeySlice = append(blsPubkeySlice, app.strategy.NextEpochValData.NextAccountMap.MapList[tmAddress].BlsKeyString)
		}
	}

	for i, v := range app.strategy.CurrHeightValData.UpdateValidators {
		pubkey, _ := tmTypes.PB2TM.PubKey(v.PubKey)
		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		signer := app.strategy.CurrHeightValData.AccountMap.MapList[tmAddress].Signer

		if topKSignerMap[signer] == nil {
			updateValidators = append(updateValidators,
				abciTypes.ValidatorUpdate{
					PubKey: app.strategy.CurrHeightValData.UpdateValidators[i].PubKey,
					Power:  int64(0),
				})
		}
	}

	app.strategy.CurrHeightValData.UpdateValidators = poolValidators
	app.strategy.CurrHeightValData.UpdateValidatorWeights = nil

	return abciTypes.ResponseEndBlock{ValidatorUpdates: updateValidators, BlsKeyString: blsPubkeySlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
}

func (app *EthermintApplication) InitPersistData() bool {
	app.logger.Info("Init Persist Data")
	// marshal map to jsonBytes,is it sorted?
	wsState, _ := app.backend.Es().State()

	var initFlag bool

	app.logger.Info("Read NextEpochValData")
	nextBytes := wsState.GetCode(common.HexToAddress("0x8888888888888888888888888888888888888888"))

	app.logger.Info("Read currentRoundValData")
	currBytes := wsState.GetCode(common.HexToAddress("0x7777777777777777777777777777777777777777"))

	app.logger.Info("Read LastUpdateValidators")
	lastValsBytes := wsState.GetCode(common.HexToAddress("0x9999999999999999999999999999999999999999"))

	if len(nextBytes) == 0 {
		// no predata existed
		app.logger.Info("no pre NextEpochValData")
	} else {
		app.logger.Info("NextEpochValData Not nil")
		err := json.Unmarshal(nextBytes, &app.strategy.NextEpochValData)
		if err != nil {
			panic("initial NextEpochValData error")
		} else {
			initFlag = true
		}
	}

	if len(currBytes) == 0 {
		// no predata existed
		app.logger.Info("no pre CurrentHeightValData")
	} else {
		app.logger.Info("CurrentHeightValData Not nil")
		err := json.Unmarshal(currBytes, &app.strategy.CurrHeightValData)
		if err != nil {
			panic("initial CurrentHeightValData error")
		} else {
			initFlag = true
		}
	}

	lastUpdateValidators := ethmintTypes.LastUpdateValidators{}
	if len(lastValsBytes) == 0 {
		// no predata existed
		app.logger.Info("no pre lastValsBytes")
	} else {
		app.logger.Info("lastValsBytes Not nil")
		err := json.Unmarshal(currBytes, &lastUpdateValidators)
		if err != nil {
			panic("initial updateValidators error")
		} else {
			app.strategy.CurrHeightValData.UpdateValidators = lastUpdateValidators.UpdateValidators
			initFlag = true
		}
	}

	return initFlag
}

func (app *EthermintApplication) SetPersistenceData(height int64) {
	app.logger.Info(fmt.Sprintf("set persist data in height %v", height))
	wsState, _ := app.getCurrentState()

	if app.strategy.NextEpochValData.ChangedFlagThisBlock || height%200 == 0 {
		nextBytes, _ := json.Marshal(app.strategy.NextEpochValData)
		wsState.SetCode(common.HexToAddress("0x8888888888888888888888888888888888888888"), nextBytes)
	}

	if height%200 == 0 {
		currBytes, _ := json.Marshal(app.strategy.CurrHeightValData)
		wsState.SetCode(common.HexToAddress("0x7777777777777777777777777777777777777777"), currBytes)
	}

	//persist every height
	lastUpdateValidators := ethmintTypes.LastUpdateValidators{app.strategy.CurrHeightValData.UpdateValidators}
	valBytes, _ := json.Marshal(lastUpdateValidators)
	wsState.SetCode(common.HexToAddress("0x9999999999999999999999999999999999999999"), valBytes)
}
