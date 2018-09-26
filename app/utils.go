package app

import (
	"bytes"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
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

func (app *EthermintApplication) AddValidatorTx(account common.Address, balance int64, address tmTypes.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	return false, nil
}

func (app *EthermintApplication) RemoveValidatorTx(account common.Address, balance int64, address tmTypes.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	return false, nil
}

func (app *EthermintApplication) UpsertPosItem(account common.Address, balance int64, address tmTypes.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	if app.strategy != nil {
		bool, err := app.strategy.PosTable.UpsertPosItem(account, balance, address, pubkey)
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
					validator := abciTypes.Validator{
						Address: app.strategy.PosTable.PosArray[(int(height)+j-1)%index].Address,
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
					validator := abciTypes.Validator{
						Address: app.strategy.PosTable.PosArray[(int(height)+i-1)%index].Address,
						PubKey:  app.strategy.PosTable.PosArray[(int(height)+i-1)%index].PubKey,
						Power:   1,
					}
					if i == 0 {
						validatorsSlice = append(validatorsSlice, validator)
						app.strategy.ValidatorSet.CurrentValidators = append(app.
							strategy.ValidatorSet.CurrentValidators, &validator)
					}else if bytes.Equal(validator.Address, validatorsSlice[2].Address) {
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
