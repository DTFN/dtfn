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
	"strings"
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
func (app *EthermintApplication) SetValidators(validators []*abciTypes.Validator) {
	if app.strategy != nil {
		app.strategy.SetValidators(validators)
	}
}

func (app *EthermintApplication) StartHttpServer() {
	go app.httpServer.HttpServer.ListenAndServe()
}

func (app *EthermintApplication) UpsertValidatorTx(signer common.Address, currentHeight *big.Int, balance *big.Int,
	beneficiary common.Address, pubkey crypto.PubKey, blsKeyString string) (bool, error) {
	app.GetLogger().Info("You are upsert ValidatorTxing")

	if pubkey == nil || len(blsKeyString) == 0 {
		app.GetLogger().Info("nil validator pubkey or bls pubkey")
		return false, errors.New("nil validator pubkey or bls pubkey")
	}
	if app.strategy != nil {
		// judge whether is a valid addValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		abciPubKey := tmTypes.TM2PB.PubKey(pubkey)

		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		existFlag := false
		for i := 0; i < len(app.strategy.NextRoundValData.NextRoundCandidateValidators); i++ {
			if bytes.Equal(pubkey.Address(), app.strategy.
				NextRoundValData.NextRoundCandidateValidators[i].Address) {
				origSigner := app.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddress].Signer
				if origSigner.String() != signer.String() {
					app.GetLogger().Info("validator was voted by another signer")
					return false, errors.New("validator was voted by another signer")
				}
				existFlag = true
			}
		}
		//做这件事之前必须确认这个signer，不是MapList中已经存在的。
		//1.signer相同，可能来作恶;  2.signer相同，可能不作恶，因为有相同maplist;  3.signer不相同
		signerExisted := false
		for _, v := range app.strategy.NextRoundValData.NextAccountMapList.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				signerExisted = true
				break
			}
		}
		blsExisted := false
		for _, v := range app.strategy.NextRoundValData.NextAccountMapList.MapList {
			if v.BlsKeyString == blsKeyString {
				blsExisted = true
				break
			}
		}

		stateDb, _ := app.getCurrentState()
		if !signerExisted && !existFlag && !blsExisted &&
			blacklist.IsLock(stateDb, currentHeight.Int64(), signer) {
			// signer不相同 signer should not be locked
			// If is a valid addValidatorTx,change the data in the strategy
			// Should change the maplist and postable and nextCandidateValidator
			upsertFlag, err := app.strategy.NextRoundValData.NextRoundPosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Info(err.Error())
				return false, err
			}
			app.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddress] = &ethmintTypes.AccountMap{
				Beneficiary:   beneficiary,
				Signer:        signer,
				SignerBalance: balance,
				BlsKeyString:  blsKeyString,
			}
			app.strategy.NextRoundValData.NextRoundCandidateValidators = append(app.
				strategy.NextRoundValData.NextRoundCandidateValidators,
				&abciTypes.Validator{
					PubKey:  abciPubKey,
					Power:   1,
					Address: pubkey.Address(),
				})
			app.GetLogger().Info("add Validator Tx success")
			app.strategy.NextRoundValData.NextRoundPosTable.ChangedFlagThisBlock = true
			return true, nil
		} else if existFlag && signerExisted && blsExisted {
			if app.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddress].BlsKeyString != blsKeyString || !bytes.
				Equal(app.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddress].Signer.Bytes(), signer.Bytes()) {
				return false, errors.New("bls or validator pubkey was signed by other people")
			}
			//同singer，同MapList[tmAddress]，是来改动balance的
			upsertFlag, err := app.strategy.NextRoundValData.NextRoundPosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Info(err.Error())
				return false, err
			}
			app.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddress].Beneficiary = beneficiary
			app.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddress].SignerBalance = balance
			app.GetLogger().Info("upsert Validator Tx success")
			app.strategy.NextRoundValData.NextRoundPosTable.ChangedFlagThisBlock = true
			return true, nil
		} else {
			//同singer，不同MapList[tmAddress]，来捣乱的
			app.GetLogger().Info("signer has voted")
			return false, errors.New("signer has voted")
		}

	}
	return false, errors.New("upsertFailed for unknown reason")
}

func (app *EthermintApplication) RemoveValidatorTx(signer common.Address) (bool, error) {
	app.GetLogger().Info("You are removeValidatorTx")
	if app.strategy != nil {
		//找到tmAddress，另这个的signer与输入相等
		var tmAddress string
		for k, v := range app.strategy.NextRoundValData.NextAccountMapList.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				tmAddress = k
				break
			}
		}

		if len(app.strategy.NextRoundValData.NextRoundCandidateValidators) <= 4 {
			app.GetLogger().Info("can not remove validator for error-tolerant")
			return false, errors.New("can not remove validator for error-tolerant")
		}

		// judge whether is a valid removeValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		//tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		existFlag := false
		markIndex := 0
		tmBytes, err := hex.DecodeString(tmAddress)
		if err != nil {
			return false, err
		}
		for i := 0; i < len(app.strategy.NextRoundValData.NextRoundCandidateValidators); i++ {
			if bytes.Equal(tmBytes, app.strategy.
				NextRoundValData.NextRoundCandidateValidators[i].Address) {
				existFlag = true
				markIndex = i
				break
			}
		}
		if existFlag {
			removeFlag, err := app.strategy.NextRoundValData.NextRoundPosTable.RemovePosItem(signer)
			if err != nil || !removeFlag {
				app.GetLogger().Info("posTable remove failed")
				return false, errors.New("posTable remove failed")
			}
			//app.strategy.CurrRoundValData.AccMapInitial.MapList[tmAddress] = app.strategy.CurrRoundValData.AccountMapList.MapList[tmAddress]
			delete(app.strategy.NextRoundValData.NextAccountMapList.MapList, tmAddress)

			var validatorPre, validatorNext []*abciTypes.Validator

			if len(app.strategy.NextRoundValData.NextRoundCandidateValidators) == 1 {
				app.strategy.NextRoundValData.NextRoundCandidateValidators = nil
			} else if markIndex == 0 {
				app.strategy.NextRoundValData.NextRoundCandidateValidators = app.strategy.
					NextRoundValData.NextRoundCandidateValidators[1:]
			} else if markIndex == len(app.strategy.NextRoundValData.NextRoundCandidateValidators)-1 {
				app.strategy.NextRoundValData.NextRoundCandidateValidators = app.strategy.NextRoundValData.
					NextRoundCandidateValidators[0 : len(app.strategy.NextRoundValData.NextRoundCandidateValidators)-1]
			} else {
				validatorPre = app.strategy.NextRoundValData.NextRoundCandidateValidators[0:markIndex]
				validatorNext = app.strategy.NextRoundValData.NextRoundCandidateValidators[markIndex+1:]
				app.strategy.NextRoundValData.NextRoundCandidateValidators = validatorPre
				for i := 0; i < len(validatorNext); i++ {
					app.strategy.NextRoundValData.NextRoundCandidateValidators = append(app.
						strategy.NextRoundValData.NextRoundCandidateValidators, validatorNext[i])
				}
			}
			//if validator is exist in the currentValidators,it must be removed
			app.GetLogger().Info("remove validatorTx success")
			app.strategy.NextRoundValData.NextRoundPosTable.ChangedFlagThisBlock = true
			return true, nil
		} else {
			app.GetLogger().Info("signer address not existed")
		}
		// If is a valid removeValidator,change the data in the strategy
	}
	return false, errors.New("app strategy nil")
}

func (app *EthermintApplication) SetThreShold(threShold *big.Int) {
	if app.strategy != nil {
		app.strategy.CurrRoundValData.PosTable.SetThreShold(threShold)
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
	return abciTypes.ResponseEndBlock{}
}

// CollectTx invokes CollectTx on the strategy
// #unstable
func (app *EthermintApplication) CollectTx(tx *types.Transaction) {
	if app.strategy != nil {
		app.strategy.CollectTx(tx)
	}
}

func (app *EthermintApplication) enterInitial(height int64) abciTypes.ResponseEndBlock {
	if len(app.strategy.CurrRoundValData.InitialValidators) == 0 {
		// There is no nextCandidateValidators for initial height
		return abciTypes.ResponseEndBlock{}
	} else {
		var validatorsSlice []abciTypes.Validator
		validators := app.strategy.GetUpdatedValidators()

		if len(app.strategy.CurrRoundValData.CurrCandidateValidators) == 0 {
			return abciTypes.ResponseEndBlock{}
		}

		maxValidators := 0
		if len(app.strategy.CurrRoundValData.CurrCandidateValidators) < 7 {
			maxValidators = len(app.strategy.CurrRoundValData.CurrCandidateValidators)
		} else {
			maxValidators = 7
		}

		votedValidators := make(map[string]bool)
		votedValidatorsIndex := make(map[string]int)

		//select validators from posTable
		for j := 0; len(validatorsSlice) != maxValidators; j++ {
			tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.CurrRoundValData.PosTable.SelectItemByHeightValue(int(height) + j - 1).PubKey)
			validator := abciTypes.Validator{
				Address: tmPubKey.Address(),
				PubKey:  app.strategy.CurrRoundValData.PosTable.SelectItemByHeightValue(int(height) + j - 1).PubKey,
				Power:   1000,
			}
			if votedValidators[tmPubKey.Address().String()] {
				validatorsSlice[votedValidatorsIndex[tmPubKey.Address().String()]].Power++
			} else {
				//Remember tmPubKey.Address 's index in the currentValidators Array
				votedValidatorsIndex[tmPubKey.Address().String()] = len(validatorsSlice)

				votedValidators[tmPubKey.Address().String()] = true
				validatorsSlice = append(validatorsSlice, validator)
				app.strategy.CurrRoundValData.CurrentValidators = append(app.
					strategy.CurrRoundValData.CurrentValidators, &validator)
			}
		}

		//record the currentValidator weight for accumulateReward
		for i := 0; i < maxValidators; i++ {
			app.strategy.CurrRoundValData.CurrentValidatorWeight = append(
				app.strategy.CurrRoundValData.CurrentValidatorWeight,
				validatorsSlice[i].Power-999)
		}

		//append the validators which will be deleted
		for i := 0; i < len(validators); i++ {
			tmPubKey, _ := tmTypes.PB2TM.PubKey(validators[i].PubKey)
			if !votedValidators[tmPubKey.Address().String()] {
				validatorsSlice = append(validatorsSlice,
					abciTypes.Validator{
						//Address : app.strategy.PosTable.SelectItemByRandomValue(int(height)).Address,
						Address: validators[i].Address,
						Power:   int64(0),
						PubKey:  validators[i].PubKey,
					})
			}
		}
		return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
	}
}

func (app *EthermintApplication) enterSelectValidators(seed []byte, height int64) abciTypes.ResponseEndBlock {

	if app.strategy.BlsSelectStrategy {
	} else {
	}

	var validatorsSlice []abciTypes.Validator
	var valCopy []abciTypes.Validator

	maxValidatorSlice := 0
	if len(app.strategy.CurrRoundValData.CurrCandidateValidators) <= 4 {
		return abciTypes.ResponseEndBlock{}
	} else if len(app.strategy.CurrRoundValData.CurrCandidateValidators) < 7 {
		maxValidatorSlice = len(app.strategy.CurrRoundValData.CurrCandidateValidators)
	} else {
		maxValidatorSlice = 7
	}

	for i := 0; i < len(app.strategy.CurrRoundValData.CurrentValidators); i++ {
		valCopy = append(valCopy, *app.strategy.CurrRoundValData.CurrentValidators[i])
	}

	app.strategy.CurrRoundValData.CurrentValidators = nil
	app.strategy.CurrRoundValData.CurrentValidatorWeight = nil

	// we use map to remember which validators voted has put into validatorSlice
	votedValidators := make(map[string]bool)
	votedValidatorsIndex := make(map[string]int)

	//select validators from posTable
	for i := 0; len(validatorsSlice) != maxValidatorSlice; i++ {
		var tmPubKey crypto.PubKey
		var validator abciTypes.Validator
		if height == -1 {
			//height=-1 表示 seed 存在，使用seed
			tmPubKey, _ = tmTypes.PB2TM.PubKey(app.strategy.CurrRoundValData.PosTable.SelectItemBySeedValue(seed, i).PubKey)
			validator = abciTypes.Validator{
				Address: tmPubKey.Address(),
				PubKey:  app.strategy.CurrRoundValData.PosTable.SelectItemBySeedValue(seed, i).PubKey,
				Power:   1000,
			}
		} else {
			//seed 不存在，使用height
			tmPubKey, _ = tmTypes.PB2TM.PubKey(app.strategy.CurrRoundValData.PosTable.SelectItemByHeightValue(int(height) + i - 1).PubKey)
			validator = abciTypes.Validator{
				Address: tmPubKey.Address(),
				PubKey:  app.strategy.CurrRoundValData.PosTable.SelectItemByHeightValue(int(height) + i - 1).PubKey,
				Power:   1000,
			}
		}

		if votedValidators[tmPubKey.Address().String()] {
			validatorsSlice[votedValidatorsIndex[tmPubKey.Address().String()]].Power++
		} else {
			//Remember tmPubKey.Address 's index in the currentValidators Array
			votedValidatorsIndex[tmPubKey.Address().String()] = len(validatorsSlice)

			votedValidators[tmPubKey.Address().String()] = true
			validatorsSlice = append(validatorsSlice, validator)
			app.strategy.CurrRoundValData.CurrentValidators = append(app.
				strategy.CurrRoundValData.CurrentValidators, &validator)
		}
	}

	//record the currentValidator weight for accumulateReward
	for i := 0; i < maxValidatorSlice; i++ {
		app.strategy.CurrRoundValData.CurrentValidatorWeight = append(
			app.strategy.CurrRoundValData.CurrentValidatorWeight,
			validatorsSlice[i].Power-999)
	}

	//append the validators which will be deleted
	for i := 0; i < len(valCopy); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(valCopy[i].PubKey)
		if !votedValidators[tmPubKey.Address().String()] {
			validatorsSlice = append(validatorsSlice,
				abciTypes.Validator{
					Address: valCopy[i].Address,
					PubKey:  valCopy[i].PubKey,
					Power:   int64(0),
				})
		}
	}
	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
}

func (app *EthermintApplication) blsValidators(height int64) abciTypes.ResponseEndBlock {

	var validatorsSlice []abciTypes.Validator
	var blsPubkeySlice []string

	app.strategy.CurrRoundValData.CurrentValidators = nil
	tmAddressMap := app.strategy.CurrRoundValData.PosTable.PosNodeSortList.GetTopValTmAddress()

	for i := 0; i < len(app.strategy.CurrRoundValData.CurrCandidateValidators); i++ {

		pubkey, _ := tmTypes.PB2TM.PubKey(app.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey)
		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))

		blsPower := big.NewInt(1)
		blsPower.Div(app.strategy.CurrRoundValData.AccountMapList.MapList[tmAddress].SignerBalance,
			app.strategy.CurrRoundValData.PosTable.Threshold)

		if tmAddressMap[tmAddress] {
			app.strategy.CurrRoundValData.CurrentValidators = append(app.
				strategy.CurrRoundValData.CurrentValidators, &abciTypes.Validator{
				Address: app.strategy.CurrRoundValData.CurrCandidateValidators[i].Address,
				PubKey:  app.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey,
				Power:   blsPower.Int64(),
			})

			validatorsSlice = append(validatorsSlice,
				abciTypes.Validator{
					Address: app.strategy.CurrRoundValData.CurrCandidateValidators[i].Address,
					PubKey:  app.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey,
					Power:   blsPower.Int64(),
				})
			blsPubkeySlice = append(blsPubkeySlice, app.strategy.CurrRoundValData.AccountMapList.MapList[tmAddress].BlsKeyString)
		}
	}

	app.strategy.CurrRoundValData.CurrentValidatorWeight = nil

	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, BlsKeyString: blsPubkeySlice}
}

func (app *EthermintApplication) InitialPos() {
	app.logger.Info("BeginBlock")
	// marshal map to jsonBytes,is it sorted?
	wsState, _ := app.backend.Es().State()
	app.logger.Info("Read accountMap")
	accountMap := wsState.GetCode(common.HexToAddress("0x7777777777777777777777777777777777777777"))
	if len(accountMap) == 0 {
		// no predata existed
	} else {
		accountmaplist := ethmintTypes.AccountMapList{}
		err := json.Unmarshal(accountMap, &accountmaplist)
		if err != nil {
			panic("initial accountmap error")
		} else {
			app.strategy.CurrRoundValData.AccountMapList = &accountmaplist
		}
	}

	app.logger.Info("Read Pos Table")
	posTable := wsState.GetCode(common.HexToAddress("0x8888888888888888888888888888888888888888"))
	if len(posTable) == 0 {
		// no predata existed
	} else {
		app.logger.Info("PosTable Not nil")
		posTableInitial := ethmintTypes.PosTable{}
		err := json.Unmarshal(posTable, &posTableInitial)
		if err != nil {
			panic("initial postable error")
		} else {
			app.strategy.CurrRoundValData.PosTable = &posTableInitial
		}
	}
}

func (app *EthermintApplication) PersistencePos() {
	app.logger.Info("EndBlock")
	if app.strategy.CurrRoundValData.PosTable.ChangedFlagThisBlock {
		app.logger.Info("PosTable has changed")
		app.strategy.CurrRoundValData.PosTable.ChangedFlagThisBlock = false
		app.logger.Info("write accountMap")
		accountMapBytes, _ := json.Marshal(app.strategy.CurrRoundValData.AccountMapList)
		posTableBytes, _ := json.Marshal(app.strategy.CurrRoundValData.PosTable)
		wsState, _ := app.backend.Es().State()
		wsState.SetCode(common.HexToAddress("0x7777777777777777777777777777777777777777"), accountMapBytes)
		wsState.SetCode(common.HexToAddress("0x8888888888888888888888888888888888888888"), posTableBytes)
	} else {
		app.logger.Info("PosTable hasn't changed")
	}
}
