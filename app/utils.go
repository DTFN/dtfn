package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
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

func (app *EthermintApplication) UpsertValidatorTx(signer common.Address, balance *big.Int,
	beneficiary common.Address, pubkey crypto.PubKey) (bool, error) {
	app.GetLogger().Info("You are upsert ValidatorTxing")
	if len(app.strategy.ValidatorSet.CornerStoneValidators) < 4 {
		app.GetLogger().Info("upsert failed for len(cornerStoneValidator) less than 5")
		return false, errors.New("Not support for upsertValidatorTx because of not enough cornerStoneValidator")
	}
	if app.strategy != nil {
		// judge whether is a valid addValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		if pubkey == nil {
			app.GetLogger().Info("nil validator pubkey")
			return false, errors.New("nil validator pubkey")
		}
		abciPubKey := tmTypes.TM2PB.PubKey(pubkey)

		for i := 0; i < len(app.strategy.ValidatorSet.CornerStoneValidators); i++ {
			if bytes.Equal(pubkey.Address(), app.strategy.
				ValidatorSet.CornerStoneValidators[i].Address) {
				app.GetLogger().Info("can not use cornerStoneValidator validator")
				return false, errors.New("can not use cornerStoneValidator validator")
			}
		}

		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		existFlag := false
		for i := 0; i < len(app.strategy.ValidatorSet.NextHeightCandidateValidators); i++ {
			if bytes.Equal(pubkey.Address(), app.strategy.
				ValidatorSet.NextHeightCandidateValidators[i].Address) {
				origSigner := app.strategy.AccountMapList.MapList[tmAddress].Signer
				if origSigner.String() != signer.String() {
					app.GetLogger().Info("validator was voted by another signer")
					return false, errors.New("can not use cornerStoneValidator validator")
				}
				existFlag = true
			}
		}
		//做这件事之前必须确认这个signer，不是MapList中已经存在的。
		//1.signer相同，可能来作恶;  2.signer相同，可能不作恶，因为有相同maplist;  3.signer不相同
		same := false
		for _, v := range app.strategy.AccountMapList.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				same = true
				break
			}
		}

		if !same && !existFlag {
			// signer不相同
			// If is a valid addValidatorTx,change the data in the strategy
			// Should change the maplist and postable and nextCandidateValidator
			upsertFlag, err := app.strategy.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Info(err.Error())
				return false, err
			}
			app.strategy.AccountMapList.MapList[tmAddress] = &tmTypes.AccountMap{
				Beneficiary:   beneficiary,
				Signer:        signer,
				SignerBalance: balance,
			}
			app.strategy.ValidatorSet.NextHeightCandidateValidators = append(app.
				strategy.ValidatorSet.NextHeightCandidateValidators,
				&abciTypes.Validator{
					PubKey:  abciPubKey,
					Power:   1,
					Address: pubkey.Address(),
				})
			app.GetLogger().Info("add Validator Tx success")
			return true, nil
		} else if existFlag && same {
			//同singer，同MapList[tmAddress]，是来改动balance的
			upsertFlag, err := app.strategy.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			if err != nil || !upsertFlag {
				app.GetLogger().Info(err.Error())
				return false, err
			}
			app.strategy.AccountMapList.MapList[tmAddress].Beneficiary = beneficiary
			app.strategy.AccountMapList.MapList[tmAddress].SignerBalance = balance
			app.GetLogger().Info("upsert Validator Tx success")
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
		for k, v := range app.strategy.AccountMapList.MapList {
			if bytes.Equal(v.Signer.Bytes(), signer.Bytes()) {
				tmAddress = k
				break
			}
		}

		//return fail if tmAddress == commitValidatorAddress
		for i := 0; i < len(app.strategy.ValidatorSet.CornerStoneValidators); i++ {
			commitValidatorAddress := hex.EncodeToString(app.strategy.ValidatorSet.
				CornerStoneValidators[i].Address)
			if commitValidatorAddress == tmAddress {
				app.GetLogger().Info("can not use cornerStoneValidator validator")
				return false, errors.New("can not use cornerStoneValidator validator")
			}
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
		for i := 0; i < len(app.strategy.ValidatorSet.NextHeightCandidateValidators); i++ {
			if bytes.Equal(tmBytes, app.strategy.
				ValidatorSet.NextHeightCandidateValidators[i].Address) {
				existFlag = true
				markIndex = i
				break
			}
		}
		if existFlag {
			removeFlag, err := app.strategy.PosTable.RemovePosItem(signer)
			if err != nil || !removeFlag {
				app.GetLogger().Info("posTable remove failed")
				return false, errors.New("posTable remove failed")
			}
			app.strategy.AccountMapListTemp.MapList[tmAddress] = app.strategy.AccountMapList.MapList[tmAddress]
			delete(app.strategy.AccountMapList.MapList, tmAddress)

			var validatorPre, validatorNext []*abciTypes.Validator

			if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) == 1 {
				app.strategy.ValidatorSet.NextHeightCandidateValidators = nil
			} else if markIndex == 0 {
				app.strategy.ValidatorSet.NextHeightCandidateValidators = app.strategy.ValidatorSet.NextHeightCandidateValidators[1:]
			} else if markIndex == len(app.strategy.ValidatorSet.NextHeightCandidateValidators)-1 {
				app.strategy.ValidatorSet.NextHeightCandidateValidators = app.strategy.ValidatorSet.
					NextHeightCandidateValidators[0 : len(app.strategy.ValidatorSet.NextHeightCandidateValidators)-1]
			} else {
				validatorPre = app.strategy.ValidatorSet.NextHeightCandidateValidators[0:markIndex]
				validatorNext = app.strategy.ValidatorSet.NextHeightCandidateValidators[markIndex+1:]
				app.strategy.ValidatorSet.NextHeightCandidateValidators = validatorPre
				for i := 0; i < len(validatorNext); i++ {
					app.strategy.ValidatorSet.NextHeightCandidateValidators = append(app.
						strategy.ValidatorSet.NextHeightCandidateValidators, validatorNext[i])
				}
			}
			//if validator is exist in the currentValidators,it must be removed
			app.GetLogger().Info("remove validatorTx success")
			return true, nil
		} else {
			app.GetLogger().Info("signer address not existed")
		}
		// If is a valid removeValidator,change the data in the strategy
	}
	return false, errors.New("app strategy nil")
}

func (app *EthermintApplication) UpsertPosItem(account common.Address, balance *big.Int, beneficiary common.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	if app.strategy != nil {
		bool, err := app.strategy.PosTable.UpsertPosItem(account, balance, beneficiary, pubkey)
		return bool, err
	}
	return false, nil
}

func (app *EthermintApplication) RemovePosItem(account common.Address) (bool, error) {
	if app.strategy != nil {
		bool, err := app.strategy.PosTable.RemovePosItem(account)
		return bool, err
	}
	return false, nil
}

func (app *EthermintApplication) SetThreShold(threShold *big.Int) {
	if app.strategy != nil {
		app.strategy.PosTable.SetThreShold(threShold)
	}
}

// GetUpdatedValidators returns an updated validator set from the strategy
// #unstable
func (app *EthermintApplication) GetUpdatedValidators(height int64) abciTypes.ResponseEndBlock {
	if app.strategy != nil {
		if int(height) == 1 {
			return app.enterInitial(height)
		} else if int(height)%200 != 0 {
			return app.enterSelectValidators(height)
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
	if len(app.strategy.ValidatorSet.InitialValidators) == 0 {
		// There is no nextCandidateValidators for initial height
		return abciTypes.ResponseEndBlock{}
	} else {
		var validatorsSlice []abciTypes.Validator
		validators := app.strategy.GetUpdatedValidators()

		for i := 0; i < len(validators); i++ {
			validatorsSlice = append(validatorsSlice,
				abciTypes.Validator{
					//Address : app.strategy.PosTable.SelectItemByRandomValue(int(height)).Address,
					Address: validators[i].Address,
					Power:   int64(0),
					PubKey:  validators[i].PubKey,
				})
		}

		for i := 0; i < len(app.strategy.ValidatorSet.CornerStoneValidators); i++ {
			validatorsSlice = append(validatorsSlice, *app.strategy.ValidatorSet.CornerStoneValidators[i])
		}

		for j := 0; j < len(app.strategy.ValidatorSet.InitialValidators); j++ {
			address := strings.ToLower(hex.EncodeToString(app.strategy.ValidatorSet.
				InitialValidators[j].Address))
			upsertFlag, _ := app.UpsertPosItem(
				app.strategy.AccountMapList.MapList[address].Signer,
				app.strategy.AccountMapList.MapList[address].SignerBalance,
				app.strategy.AccountMapList.MapList[address].Beneficiary,
				app.strategy.ValidatorSet.InitialValidators[j].PubKey)
			if upsertFlag {
				app.strategy.ValidatorSet.NextHeightCandidateValidators = append(app.
					strategy.ValidatorSet.NextHeightCandidateValidators, app.
					strategy.ValidatorSet.InitialValidators[j])
			} else {
				tmAddress := hex.EncodeToString(app.strategy.ValidatorSet.
					InitialValidators[j].Address)
				delete(app.strategy.AccountMapList.MapList, tmAddress)
				app.GetLogger().Info("remove not enough balance validators")
			}
		}
		if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) == 0 {
			return abciTypes.ResponseEndBlock{}
		}

		// if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) >0
		// len(app.strategy.ValidatorSet.InitialValidators) must >0
		// so len(app.strategy.ValidatorSet.InitialValidators) must = 4
		// while we are initingChain,we process the following code
		// if len(validators) > 4 {
		//	app.strategy.ValidatorSet.CornerStoneValidators = validators[0:4]
		//	app.strategy.ValidatorSet.InitialValidators = validators[4:]
		//}

		// and maxValidators must <= 7 and >= 4
		// if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) >=3
		// maxValidators =7
		maxValidators := 0
		if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) < 3 {
			maxValidators = len(app.strategy.ValidatorSet.NextHeightCandidateValidators) +
				len(app.strategy.ValidatorSet.CornerStoneValidators)
		} else {
			maxValidators = 7
		}

		// len(validators) = all initial validators write in the genesis.json file
		// maxValidators = next height validators who take part in consensus
		// all validators has been put into the validatorSlice,
		// we just need to put the voted validator into validatorSlice
		// we use map to remember which validators voted has put into validatorSlice
		votedValidators := make(map[string]bool)

		for j := 0; len(validatorsSlice) != maxValidators+len(validators); j++ {
			tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.PosTable.SelectItemByRandomValue(int(height) + j - 1).PubKey)
			validator := abciTypes.Validator{
				Address: tmPubKey.Address(),
				PubKey:  app.strategy.PosTable.SelectItemByRandomValue(int(height) + j - 1).PubKey,
				Power:   1,
			}
			if votedValidators[tmPubKey.Address().String()] {
				// existed,do nothing
			} else {
				votedValidators[tmPubKey.Address().String()] = true
				validatorsSlice = append(validatorsSlice, validator)
				app.strategy.ValidatorSet.CurrentValidators = append(app.
					strategy.ValidatorSet.CurrentValidators, &validator)
			}
		}
		return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
	}
}

func (app *EthermintApplication) enterSelectValidators(height int64) abciTypes.ResponseEndBlock {

	var validatorsSlice []abciTypes.Validator
	for i := 0; i < len(app.strategy.ValidatorSet.CurrentValidators); i++ {
		validatorsSlice = append(validatorsSlice,
			abciTypes.Validator{
				Address: app.strategy.ValidatorSet.CurrentValidators[i].Address,
				PubKey:  app.strategy.ValidatorSet.CurrentValidators[i].PubKey,
				Power:   int64(0),
			})
	}

	// app.strategy.ValidatorSet.CurrentValidators not include cornerStoneValidators
	// len(app.strategy.ValidatorSet.NextHeightCandidateValidators) == 0
	// we should just remove all the currentValidators

	// if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) >= 3
	// we must keep the maxValidatorSlice = 7 (cornerStoneValidator should equal 4)
	// cornerStoneValidator should not be remove by removeValidatorTx
	// and all the upsertValidatorTx should be failed when len(cornerStoneValidator) < 4
	//

	maxValidatorSlice := 0
	if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) == 0 {
		app.strategy.ValidatorSet.CurrentValidators = nil
		return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
	} else if len(app.strategy.ValidatorSet.NextHeightCandidateValidators) < 3 {
		maxValidatorSlice = len(app.strategy.ValidatorSet.NextHeightCandidateValidators) +
			len(app.strategy.ValidatorSet.CurrentValidators)
	} else {
		maxValidatorSlice = 3 + len(app.strategy.ValidatorSet.CurrentValidators)
	}
	app.strategy.ValidatorSet.CurrentValidators = nil

	// we use map to remember which validators voted has put into validatorSlice
	votedValidators := make(map[string]bool)

	for i := 0; len(validatorsSlice) != maxValidatorSlice; i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.PosTable.SelectItemByRandomValue(int(height) + i - 1).PubKey)
		validator := abciTypes.Validator{
			Address: tmPubKey.Address(),
			PubKey:  app.strategy.PosTable.SelectItemByRandomValue(int(height) + i - 1).PubKey,
			Power:   1,
		}
		if votedValidators[tmPubKey.Address().String()] {
			// existed,do nothing
		} else {
			votedValidators[tmPubKey.Address().String()] = true
			validatorsSlice = append(validatorsSlice, validator)
			app.strategy.ValidatorSet.CurrentValidators = append(app.
				strategy.ValidatorSet.CurrentValidators, &validator)
		}
	}
	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}

}

func (app *EthermintApplication) blsValidators(height int64) abciTypes.ResponseEndBlock {

	var validatorsSlice []abciTypes.Validator
	for i := 0; i < len(app.strategy.ValidatorSet.CurrentValidators); i++ {
		validatorsSlice = append(validatorsSlice,
			abciTypes.Validator{
				Address: app.strategy.ValidatorSet.CurrentValidators[i].Address,
				PubKey:  app.strategy.ValidatorSet.CurrentValidators[i].PubKey,
				Power:   int64(0),
			})
	}

	app.strategy.ValidatorSet.CurrentValidators = nil

	for i := 0; i < len(app.strategy.ValidatorSet.CornerStoneValidators); i++ {
		validatorsSlice = append(validatorsSlice,
			abciTypes.Validator{
				Address: app.strategy.ValidatorSet.CornerStoneValidators[i].Address,
				PubKey:  app.strategy.ValidatorSet.CornerStoneValidators[i].PubKey,
				Power:   int64(1),
			})
	}

	for i := 0; i < len(app.strategy.ValidatorSet.NextHeightCandidateValidators); i++ {
		app.strategy.ValidatorSet.CurrentValidators = append(app.
			strategy.ValidatorSet.CurrentValidators, &abciTypes.Validator{
			Address: app.strategy.ValidatorSet.NextHeightCandidateValidators[i].Address,
			PubKey:  app.strategy.ValidatorSet.NextHeightCandidateValidators[i].PubKey,
			Power:   int64(1),
		})

		validatorsSlice = append(validatorsSlice,
			abciTypes.Validator{
				Address: app.strategy.ValidatorSet.NextHeightCandidateValidators[i].Address,
				PubKey:  app.strategy.ValidatorSet.NextHeightCandidateValidators[i].PubKey,
				Power:   int64(1),
			})
	}

	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
}
