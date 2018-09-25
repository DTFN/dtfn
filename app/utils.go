package app

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	abciTypes "github.com/tendermint/tendermint/abci/types"
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

// GetUpdatedValidators returns an updated validator set from the strategy
// #unstable
func (app *EthermintApplication) GetUpdatedValidators(height int64) abciTypes.ResponseEndBlock {
	if app.strategy != nil {
		if len(app.committeeValidators) < 5 {
			return abciTypes.ResponseEndBlock{}
		} else if len(app.candidateValidators) <= 2 && height < 200 {
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
							Address: validators[i].Address,
							Power:   int64(0),
							PubKey:  validators[i].PubKey,
						})
				}
				for i := 0; i < 5; i++ {
					validatorsSlice = append(validatorsSlice, *app.committeeValidators[i])
				}
				for j := 0; j < 2; j++ {
					validatorsSlice = append(validatorsSlice,
						*app.candidateValidators[(int(height)%(len(app.candidateValidators)-1))+j])
					app.currentValidators = append(app.currentValidators,
						app.candidateValidators[(int(height)%(len(app.candidateValidators)-1))+j])
				}

				return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
			} else {
				for i := 0; i < 2; i++ {
					fmt.Println(len(app.currentValidators))
					validatorsSlice = append(validatorsSlice,
						abciTypes.Validator{
							Address: app.currentValidators[i].Address,
							PubKey:  app.currentValidators[i].PubKey,
							Power:   int64(0),
						})
				}
				app.currentValidators = nil
				for i := 0; i < 2; i++ {
					validatorsSlice = append(validatorsSlice,
						*app.candidateValidators[(int(height)%(len(app.candidateValidators)-1))+i])
					app.currentValidators = append(app.currentValidators,
						app.candidateValidators[(int(height)%(len(app.candidateValidators)-1))+i])
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
