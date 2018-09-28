package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tmTypes "github.com/tendermint/tendermint/types"
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

func (app *EthermintApplication) UpsertValidatorTx(signer common.Address, balance int64,
	beneficiary common.Address, pubkey crypto.PubKey) (bool, error) {
	if app.strategy != nil {
		// judge whether is a valid addValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		abciPubKey := tmTypes.TM2PB.PubKey(pubkey)

		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		existFlag := false
		for i := 0; i < len(app.strategy.ValidatorSet.NextCandidateValidators); i++ {
			if bytes.Equal(pubkey.Address(), app.strategy.
				ValidatorSet.NextCandidateValidators[i].Address) {
				origSigner := app.strategy.AccountMapList.MapList[tmAddress].Signer
				if origSigner.String() != signer.String() {
					return false, nil
				}
				existFlag = true
			}
		}
		//做这件事之前必须确认这个signer，不是MapList中已经存在的。
		//1.signer相同，可能来作恶;  2.signer相同，可能不作恶，因为有相同maplist;  3.signer不相同
		same := false
		for _,v:=range app.strategy.AccountMapList.MapList{
			if v.Signer==signer{
				same=true
			}
		}

		if same==false{
			// signer不相同
			// If is a valid addValidatorTx,change the data in the strategy
			// Should change the maplist and postable and nextCandidateValidator
			app.strategy.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
			app.strategy.AccountMapList.MapList[tmAddress] = &tmTypes.AccountMap{
				Beneficiary: beneficiary,
				Signer:      signer,
			}
			if !existFlag {
				app.strategy.ValidatorSet.NextCandidateValidators = append(app.
					strategy.ValidatorSet.NextCandidateValidators,
					&abciTypes.Validator{
						PubKey:  abciPubKey,
						Power:   1,
						Address: pubkey.Address(),
					})
			}
		}else{
			if existFlag{
				//同singer，同MapList[tmAddress]，是来改动balance的
				app.strategy.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
				app.strategy.AccountMapList.MapList[tmAddress] = &tmTypes.AccountMap{
					Beneficiary: beneficiary,
					Signer:      signer,
				}
			}else{
				//同singer，不同MapList[tmAddress]，来捣乱的
				return false, nil
			}
		}
	}
	return false, nil
}

func (app *EthermintApplication) RemoveValidatorTx(signer common.Address) (bool, error) {
	//找到tmAddress，另这个的signer与输入相等
	var tmAddress string
	for k,v:=range app.strategy.AccountMapList.MapList{
		if v.Signer == signer{
			tmAddress=k
			break
		}
	}
	if app.strategy != nil {
		// judge whether is a valid removeValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		//tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		existFlag := false
		markIndex := 0
		for i := 0; i < len(app.strategy.ValidatorSet.NextCandidateValidators); i++ {
			//pubkey.Address() 如何从tmAddress反变换回pubkey.Address()？
			if bytes.Equal([]byte(tmAddress), app.strategy.
				ValidatorSet.NextCandidateValidators[i].Address) {
				existFlag = true
				markIndex = i
				break
			}
		}
		if existFlag {
			app.strategy.PosTable.RemovePosItem(signer)
			delete(app.strategy.AccountMapList.MapList, tmAddress)
			validatorPre := app.strategy.ValidatorSet.NextCandidateValidators[0:markIndex]
			validatorNext := app.strategy.ValidatorSet.NextCandidateValidators[markIndex+1:]
			app.strategy.ValidatorSet.NextCandidateValidators = validatorPre
			for i := 0; i < len(validatorNext); i++ {
				app.strategy.ValidatorSet.NextCandidateValidators = append(app.
					strategy.ValidatorSet.NextCandidateValidators, validatorNext[i])
			}
			return false, nil
		}
		// If is a valid removeValidator,change the data in the strategy
	}
	return false, nil
}

func (app *EthermintApplication) UpsertPosItem(account common.Address, balance int64, beneficiary common.Address,
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

func (app *EthermintApplication) SetThreShold(threShold int64) {
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
		} else if height%200 != 0 {
			return app.enterSelectValidators(height)
		} else {
			return abciTypes.ResponseEndBlock{}
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
	if len(app.strategy.ValidatorSet.NextCandidateValidators) == 0 {
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
		for i := 0; i < len(app.strategy.ValidatorSet.CommitteeValidators); i++ {
			validatorsSlice = append(validatorsSlice, *app.strategy.ValidatorSet.CommitteeValidators[i])
		}

		maxValidators := 0
		if len(app.strategy.ValidatorSet.NextCandidateValidators) < 2 {
			maxValidators = 6
		} else {
			maxValidators = 7
		}

		for j := 0; len(validatorsSlice) != maxValidators+len(validators); j++ {
			tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.PosTable.SelectItemByRandomValue(int(height) + j - 1).PubKey)
			validator := abciTypes.Validator{
				Address: tmPubKey.Address(),
				PubKey:  app.strategy.PosTable.SelectItemByRandomValue(int(height) + j - 1).PubKey,
				Power:   1,
			}
			if j == 0 {
				validatorsSlice = append(validatorsSlice, validator)
				app.strategy.ValidatorSet.CurrentValidators = append(app.
					strategy.ValidatorSet.CurrentValidators, &validator)
			} else if bytes.Equal(validator.Address, validatorsSlice[5+len(validators)].Address) {
				validatorsSlice[5+len(validators)].Power++
			} else {
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

	maxValidatorSlice := 0
	if len(app.strategy.ValidatorSet.NextCandidateValidators) < 2 {
		maxValidatorSlice = 1 + len(app.strategy.ValidatorSet.CurrentValidators)
	} else {
		maxValidatorSlice = 2 + len(app.strategy.ValidatorSet.CurrentValidators)
	}
	app.strategy.ValidatorSet.CurrentValidators = nil

	for i := 0; len(validatorsSlice) != maxValidatorSlice; i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.PosTable.SelectItemByRandomValue(int(height) + i - 1).PubKey)
		validator := abciTypes.Validator{
			Address: tmPubKey.Address(),
			PubKey:  app.strategy.PosTable.SelectItemByRandomValue(int(height) + i - 1).PubKey,
			Power:   1,
		}
		if i == 0 {
			validatorsSlice = append(validatorsSlice, validator)
			app.strategy.ValidatorSet.CurrentValidators = append(app.
				strategy.ValidatorSet.CurrentValidators, &validator)
		} else if bytes.Equal(validator.Address, validatorsSlice[maxValidatorSlice-2].Address) {
			validatorsSlice[maxValidatorSlice-2].Power++
		} else {
			validatorsSlice = append(validatorsSlice, validator)
			app.strategy.ValidatorSet.CurrentValidators = append(app.
				strategy.ValidatorSet.CurrentValidators, &validator)
		}
	}
	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
}
