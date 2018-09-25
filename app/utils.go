package app

import (
	"bytes"
	"encoding/json"

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
		if height == 1 {
			var validatorsSlice []abciTypes.Validator
			validators := app.strategy.GetUpdatedValidators()
			for i := 0; i < len(validators); i++ {
				validatorsSlice = append(validatorsSlice,
					abciTypes.Validator{
						Address: validators[i].Address,
						Power:   int64(0),
						PubKey:  validators[i].PubKey,
					})
			}
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[0])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[1])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[2])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[3])
			return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
		} else if height > 1 && int(height)%3 == 1 {
			var validatorsSlice []abciTypes.Validator
			for i := 0; i < 4; i++ {
				validatorsSlice = append(validatorsSlice,
					abciTypes.Validator{
						Address: app.candidateValidators[i+2].Address,
						Power:   int64(0),
						PubKey:  app.candidateValidators[i+2].PubKey,
					})
			}
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[0])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[1])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[2])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[3])
			return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
		} else if height > 1 && int(height)%3 == 2 {
			var validatorsSlice []abciTypes.Validator
			for i := 0; i < 4; i++ {
				validatorsSlice = append(validatorsSlice,
					abciTypes.Validator{
						Address: app.candidateValidators[i].Address,
						Power:   int64(0),
						PubKey:  app.candidateValidators[i].PubKey,
					})
			}
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[1])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[2])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[3])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[4])
			return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
		} else if height > 1 && int(height)%3 == 0 {
			var validatorsSlice []abciTypes.Validator
			for i := 0; i < 4; i++ {
				validatorsSlice = append(validatorsSlice,
					abciTypes.Validator{
						Address: app.candidateValidators[i+1].Address,
						Power:   int64(0),
						PubKey:  app.candidateValidators[i+1].PubKey,
					})
			}
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[2])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[3])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[4])
			validatorsSlice = append(validatorsSlice, *app.candidateValidators[5])
			return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice}
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
