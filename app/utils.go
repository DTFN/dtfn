package app

import (
	"bytes"
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
		app.GetLogger().Info("nil validator pubKey or bls pubKey")
		return false, errors.New("nil validator pubKey or bls pubKey")
	}
	if app.strategy != nil {
		height, existFlag := app.strategy.NextEpochValData.UnBondAccountMap[signer]
		if existFlag {
			app.GetLogger().Info(fmt.Sprintf("cannot UpsertValidatorTx for signer %v already in unbondMap. You need to wait %v blocks to re-execute",
				signer, (height/ethmintTypes.EpochBlocks+ethmintTypes.UnbondedEpochs)*ethmintTypes.EpochBlocks-app.strategy.CurrHeightValData.Height))
		}

		// judge whether is a valid addValidator Tx
		// It is better to use NextCandidateValidator but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		abciPubKey := tmTypes.TM2PB.PubKey(pubKey)

		tmAddress := pubKey.Address().String()
		app.GetLogger().Info("blsKeyString: " + blsKeyString)
		app.GetLogger().Info("tmAddress: " + tmAddress)

		signerExisted := false
		blsExisted := false

		existValidator, existFlag := app.strategy.NextEpochValData.CandidateValidators[tmAddress]

		if existFlag {
			origSigner := app.strategy.NextEpochValData.AccountMap.MapList[tmAddress].Signer
			if origSigner.String() != signer.String() {
				app.GetLogger().Info("validator was voted by another signer")
				return false, errors.New("validator was voted by another signer")
			}
			if app.strategy.NextEpochValData.AccountMap.MapList[tmAddress].BlsKeyString != blsKeyString {
				return false, errors.New("bls pubKey has been used by other people")
			}
		}
		//做这件事之前必须确认这个signer，不是MapList中已经存在的。
		//1.signer相同，可能来作恶;  2.signer相同，可能不作恶，因为有相同maplist;  3.signer不相同

		for _, v := range app.strategy.NextEpochValData.AccountMap.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				signerExisted = true
				break
			}
		}

		for _, v := range app.strategy.NextEpochValData.AccountMap.MapList {
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
			upsertFlag, err := app.strategy.NextEpochValData.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Info(err.Error())
				return false, err
			}
			app.strategy.NextEpochValData.AccountMap.MapList[tmAddress] = &ethmintTypes.AccountMapItem{
				Beneficiary:  beneficiary,
				Signer:       signer,
				BlsKeyString: blsKeyString,
			}
			app.strategy.NextEpochValData.CandidateValidators[tmAddress] =
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
			upsertFlag, err := app.strategy.NextEpochValData.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Error(err.Error())
				return false, err
			}
			app.strategy.NextEpochValData.AccountMap.MapList[tmAddress].Beneficiary = beneficiary
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

func (app *EthermintApplication) PendingRemoveValidatorTx(signer common.Address) (bool, error) {
	app.GetLogger().Info(fmt.Sprintf("PendingRemoveValidatorTx. signer=%X", signer))
	if app.strategy != nil {
		if len(app.strategy.NextEpochValData.CandidateValidators)-len(app.strategy.NextEpochValData.UnBondAccountMap) <= 4 { //ensure safety
			app.GetLogger().Info("cannot remove validator for consensus safety")
			return false, errors.New("cannot remove validator for consensus safety")
		}

		height, existFlag := app.strategy.NextEpochValData.UnBondAccountMap[signer]
		if existFlag {
			app.GetLogger().Info(fmt.Sprintf("cannot remove validator for signer %v already in unbondMap. You need to wait %v blocks to re-execute",
				signer, (height/ethmintTypes.EpochBlocks+ethmintTypes.UnbondedEpochs)*ethmintTypes.EpochBlocks-app.strategy.CurrHeightValData.Height))
		}

		var tmAddress string
		for k, v := range app.strategy.NextEpochValData.AccountMap.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				tmAddress = k
				existFlag = true
				break
			}
		}
		if !existFlag {
			return false, errors.New(fmt.Sprintf("signer %v not exist in AccountMap", signer))
		}

		// judge whether is a valid removeValidator Tx
		// It is better to use NextCandidateValidator but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height

		//tmAddress := pubkey.Address().String
		_, ok := app.strategy.NextEpochValData.CandidateValidators[tmAddress]

		if !ok {
			panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in CandidateValidators", signer))
		}

		posItem, ok := app.strategy.NextEpochValData.PosTable.PosItemMap[signer]
		if !ok {
			panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in PosTable", signer))
		}
		posItem.Unbonded = true
		app.strategy.NextEpochValData.UnBondAccountMap[signer] = app.strategy.CurrHeightValData.Height
		//if validator is exist in the currentValidators,it must be removed
		app.GetLogger().Info(fmt.Sprintf("pending remove validatorTx for %X success", signer))
		app.strategy.NextEpochValData.ChangedFlagThisBlock = true
		return true, nil
	}
	return false, errors.New("app strategy nil")
}

func (app *EthermintApplication) TryRemoveValidatorTxs() (bool, error) {
	if app.strategy != nil {
		if len(app.strategy.NextEpochValData.CandidateValidators)-len(app.strategy.NextEpochValData.UnBondAccountMap) <= 4 { //ensure safety
			panic("cannot remove validator for consensus safety")
		}

		count := 0
		for signer, height := range app.strategy.NextEpochValData.UnBondAccountMap {
			if (height/ethmintTypes.EpochBlocks+ethmintTypes.UnbondedEpochs)*ethmintTypes.EpochBlocks <= app.strategy.CurrHeightValData.Height {
				result, err := app.RemoveValidatorTx(signer)
				if result && err == nil {
					count++
				}
			}
		}
		app.GetLogger().Info(fmt.Sprintf("total remove %d Validators.", count))
		return true, nil
	}
	return false, errors.New("app strategy nil")
}

func (app *EthermintApplication) RemoveValidatorTx(signer common.Address) (bool, error) {
	app.GetLogger().Info(fmt.Sprintf("removeValidatorTx. signer=%X", signer))
	var tmAddress string
	existFlag := false
	for k, v := range app.strategy.NextEpochValData.AccountMap.MapList {
		if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
			tmAddress = k
			existFlag = true
			break
		}
	}
	if !existFlag {
		panic(fmt.Sprintf("signer %v not exist in AccountMap", signer))
	}
	delete(app.strategy.NextEpochValData.AccountMap.MapList, tmAddress)

	_, ok := app.strategy.NextEpochValData.CandidateValidators[tmAddress]
	if !ok {
		panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in CandidateValidators", signer))
	}
	delete(app.strategy.NextEpochValData.CandidateValidators, tmAddress)

	removeFlag, err := app.strategy.NextEpochValData.PosTable.RemovePosItem(signer)
	if err != nil || !removeFlag {
		panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in posTable", signer))
	}

	_, ok = app.strategy.NextEpochValData.UnBondAccountMap[signer]
	if !ok {
		panic(fmt.Sprintf("Signer %v can be found in AccountMap but not found in UnBondAccountMap", signer))
	}
	delete(app.strategy.NextEpochValData.UnBondAccountMap, signer)
	//if validator is exist in the currentValidators,it must be removed
	app.GetLogger().Info(fmt.Sprintf("remove validatorTx for %X success", signer))
	app.strategy.NextEpochValData.ChangedFlagThisBlock = true
	return true, nil
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
		if int(height)%ethmintTypes.EpochBlocks != 0 {
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

	selectCount := 7 //currently fixed
	poolLen := len(app.strategy.CurrHeightValData.CurrCandidateValidators)
	if poolLen < 7 {
		app.GetLogger().Info(fmt.Sprintf("validator pool len < 7, current len %v", poolLen))
	}

	// we use map to remember which validators selected has put into validatorSlice
	selectedValidators := make(map[string]int)

	//select validators from posTable
	for i := 0; i < selectCount; i++ {
		var tmPubKey crypto.PubKey
		var validator ethmintTypes.Validator
		var signer common.Address
		var pubKey abciTypes.PubKey
		var posItem ethmintTypes.PosItem
		if height == -1 {
			//height=-1 表示 seed 存在，使用seed
			signer, posItem = app.strategy.CurrHeightValData.PosTable.SelectItemBySeedValue(seed, i)
		} else {
			//seed 不存在，使用height
			startIndex := height
			signer, posItem = app.strategy.CurrHeightValData.PosTable.SelectItemByHeightValue(startIndex + int64(i))
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
			app.strategy.CurrHeightValData.CurrentValidators[tmAddress] = validator
		}
	}
	//append the validators which will be deleted
	for address, v := range app.strategy.CurrHeightValData.CurrentValidators {
		//tmPubKey, _ := tmTypes.PB2TM.PubKey(v.PubKey)
		index, selected := selectedValidators[address]
		if selected {
			v.Power = validatorsSlice[index].Power //no use
		} else {
			validatorsSlice = append(validatorsSlice, abciTypes.ValidatorUpdate{
				PubKey: v.PubKey,
				Power:  0,
			})
			delete(app.strategy.CurrHeightValData.CurrentValidators, address)
		}
	}

	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
}

func (app *EthermintApplication) blsValidators(height int64) abciTypes.ResponseEndBlock {
	blsPubkeySlice := []string{}
	validatorsSlice := []abciTypes.ValidatorUpdate{}
	topKSignerMap := app.strategy.NextEpochValData.PosTable.TopKPosItem(100)

	for _, validator := range app.strategy.NextEpochValData.CandidateValidators {

		pubkey, _ := tmTypes.PB2TM.PubKey(validator.PubKey)
		tmAddress := pubkey.Address().String()
		signer := app.strategy.NextEpochValData.AccountMap.MapList[tmAddress].Signer

		posItem, ok := topKSignerMap[signer]
		if ok && posItem.Unbonded == false {
			power := big.NewInt(1)
			signBalance := posItem.Balance
			power.Div(signBalance,
				app.strategy.NextEpochValData.PosTable.Threshold)
			validator := abciTypes.ValidatorUpdate{
				PubKey: app.strategy.NextEpochValData.CandidateValidators[tmAddress].PubKey,
				Power:  power.Int64(),
			}
			emtValidator := ethmintTypes.Validator{validator, signer}
			app.strategy.CurrHeightValData.CurrentValidators[tmAddress] = emtValidator
			validatorsSlice = append(validatorsSlice, validator)
			blsPubkeySlice = append(blsPubkeySlice, app.strategy.NextEpochValData.AccountMap.MapList[tmAddress].BlsKeyString)
		}
	}

	for _, v := range app.strategy.CurrHeightValData.CurrentValidators {
		//pubkey, _ := tmTypes.PB2TM.PubKey(v.PubKey)
		signer := v.Signer

		_, ok := topKSignerMap[signer]
		if !ok {
			validatorsSlice = append(validatorsSlice,
				abciTypes.ValidatorUpdate{
					PubKey: v.PubKey,
					Power:  int64(0),
				})
		}
	}

	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, BlsKeyString: blsPubkeySlice, AppVersion: app.strategy.HFExpectedData.BlockVersion}
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

	app.logger.Info("Read LastCurrentValidators")
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

	var currentValidators ethmintTypes.CurrentValidators
	if len(lastValsBytes) == 0 {
		// no predata existed
		app.logger.Info("no pre lastValsBytes")
	} else {
		app.logger.Info("lastValsBytes Not nil")
		err := json.Unmarshal(lastValsBytes, &currentValidators)
		if err != nil {
			panic("initial CurrentValidators error")
		} else {
			app.strategy.CurrHeightValData.CurrentValidators = currentValidators.Validators
			initFlag = true
		}
	}

	return initFlag
}

func (app *EthermintApplication) SetPersistenceData() {

	wsState, _ := app.getCurrentState()
	height := app.strategy.CurrHeightValData.Height
	if height%ethmintTypes.EpochBlocks == 0 && height != 0 { //height==0 is when initChain calls this func
		app.TryRemoveValidatorTxs()
		//DeepCopy
		app.strategy.CurrHeightValData.AccountMap = app.strategy.NextEpochValData.AccountMap.Copy()
		app.strategy.CurrHeightValData.CurrCandidateValidators = app.strategy.NextEpochValData.ExportCandidateValidators()
		app.strategy.CurrHeightValData.PosTable = app.strategy.NextEpochValData.PosTable.Copy()
	}
	app.logger.Info(fmt.Sprintf("set persist data in height %v", height))
	if app.strategy.NextEpochValData.ChangedFlagThisBlock || height%ethmintTypes.EpochBlocks == 0 {
		nextBytes, _ := json.Marshal(app.strategy.NextEpochValData)
		wsState.SetCode(common.HexToAddress("0x8888888888888888888888888888888888888888"), nextBytes)
	}

	if height%ethmintTypes.EpochBlocks == 0 {
		currBytes, _ := json.Marshal(app.strategy.CurrHeightValData)
		wsState.SetCode(common.HexToAddress("0x7777777777777777777777777777777777777777"), currBytes)
	}

	//persist every height
	lastCurrentValidators := ethmintTypes.CurrentValidators{app.strategy.CurrHeightValData.CurrentValidators}
	valBytes, _ := json.Marshal(lastCurrentValidators)
	wsState.SetCode(common.HexToAddress("0x9999999999999999999999999999999999999999"), valBytes)
}
