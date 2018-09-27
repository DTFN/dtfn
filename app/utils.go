package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		// If is a valid addValidatorTx,change the data in the strategy
		// Should change the maplist and postable and nextCandidateValidator
		app.strategy.PosTable.UpsertPosItem(signer, balance, beneficiary, abciPubKey)
		fmt.Println(tmAddress)
		fmt.Println(beneficiary)
		fmt.Println(len(app.strategy.AccountMapList.MapList))
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
	}
	return false, nil
}

func (app *EthermintApplication) RemoveValidatorTx(signer common.Address, balance int64,
	beneficiary common.Address, pubkey crypto.PubKey) (bool, error) {
	if app.strategy != nil {
		// judge whether is a valid removeValidator Tx
		// It is better to use NextCandidateValidators but not CandidateValidators
		// because candidateValidator will changed only at (height%200==0)
		// but NextCandidateValidator will changed every height
		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		existFlag := false
		markIndex := 0
		for i := 0; i < len(app.strategy.ValidatorSet.NextCandidateValidators); i++ {
			if bytes.Equal(pubkey.Address(), app.strategy.
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
		if len(app.strategy.ValidatorSet.CommitteeValidators) < 5 {
			return abciTypes.ResponseEndBlock{}
		} else if len(app.strategy.ValidatorSet.NextCandidateValidators) <= 2 && height < 200 {
			// height=1 ,all validators <0 ,doing nothing, bls not initialed
			// height < 200 ,for test
			return abciTypes.ResponseEndBlock{}
		} else if height < 200 {
			// len(candidate validators) >2 ,use height as random value.
			var validatorsSlice []abciTypes.Validator
			validators := app.strategy.GetUpdatedValidators()
			if height == 1 {
				for i := 0; i < len(validators); i++ {
					validatorsSlice = append(validatorsSlice,
						abciTypes.Validator{
							//Address : app.strategy.PosTable.SelectItemByRandomValue(int(height)).Address,
							Address: validators[i].Address,
							Power:   int64(0),
							PubKey:  validators[i].PubKey,
						})
				}
				for i := 0; i < 5; i++ {
					validatorsSlice = append(validatorsSlice, *app.strategy.ValidatorSet.CommitteeValidators[i])
				}

				for j := 0; len(validatorsSlice) != 7+len(validators); j++ {
					index := app.strategy.PosTable.PosArraySize
					tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.PosTable.PosArray[(int(height)+j-1)%index].PubKey)
					validator := abciTypes.Validator{
						Address: tmPubKey.Address(),
						PubKey:  app.strategy.PosTable.PosArray[(int(height)+j-1)%index].PubKey,
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
			} else {
				for i := 0; i < 2; i++ {
					validatorsSlice = append(validatorsSlice,
						abciTypes.Validator{
							Address: app.strategy.ValidatorSet.CurrentValidators[i].Address,
							PubKey:  app.strategy.ValidatorSet.CurrentValidators[i].PubKey,
							Power:   int64(0),
						})
				}
				app.strategy.ValidatorSet.CurrentValidators = nil
				for i := 0; len(validatorsSlice) != 4; i++ {
					index := app.strategy.PosTable.PosArraySize
					tmPubKey, _ := tmTypes.PB2TM.PubKey(app.strategy.PosTable.PosArray[(int(height)+i-1)%index].PubKey)
					validator := abciTypes.Validator{
						Address: tmPubKey.Address(),
						PubKey:  app.strategy.PosTable.PosArray[(int(height)+i-1)%index].PubKey,
						Power:   1,
					}
					if i == 0 {
						validatorsSlice = append(validatorsSlice, validator)
						app.strategy.ValidatorSet.CurrentValidators = append(app.
							strategy.ValidatorSet.CurrentValidators, &validator)
					} else if bytes.Equal(validator.Address, validatorsSlice[2].Address) {
						validatorsSlice[2].Power++
					} else {
						validatorsSlice = append(validatorsSlice, validator)
						app.strategy.ValidatorSet.CurrentValidators = append(app.
							strategy.ValidatorSet.CurrentValidators, &validator)
					}
				}
				return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
			}
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
