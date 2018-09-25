package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/log"
	"github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"reflect"
)

// MinerRewardStrategy is a mining strategy
type MinerRewardStrategy interface {
	Receiver() common.Address
}

// ValidatorsStrategy is a validator strategy
type ValidatorsStrategy interface {
	SetValidators(validators []*types.Validator)
	CollectTx(tx *ethTypes.Transaction)
	GetUpdatedValidators() []*types.Validator
}

// Strategy encompasses all available strategies
type Strategy struct {
	MinerRewardStrategy
	ValidatorsStrategy
	currentValidators []*types.Validator
	AccountMapList *tmTypes.AccountMapList
	ValidatorTmAddress string
}

func NewStrategy() *Strategy {
	return &Strategy{}
}

// Receiver returns which address should receive the mining reward
func (s *Strategy) Receiver() common.Address {
	if s.ValidatorTmAddress == ""{
		return common.HexToAddress("0000000000000000000000000000000000000002")
	}else{
		return s.AccountMapList.MapList[s.ValidatorTmAddress].Beneficiary
	}
}

// SetValidators updates the current validators
func (strategy *Strategy) SetValidators(validators []*types.Validator) {
	strategy.currentValidators = validators
}

// CollectTx collects the rewards for a transaction
func (strategy *Strategy) CollectTx(tx *ethTypes.Transaction) {
	if reflect.DeepEqual(tx.To(), common.HexToAddress("0000000000000000000000000000000000000001")) {
		log.Info("Adding validator", "data", tx.Data())
		pubKey := types.PubKey{Data: tx.Data()}
		strategy.currentValidators = append(
			strategy.currentValidators,
			&types.Validator{
				PubKey: pubKey,
				Power:  tx.Value().Int64(),
			},
		)
	}
}

// GetUpdatedValidators returns the current validators
func (strategy *Strategy) GetUpdatedValidators() []*types.Validator {
	return strategy.currentValidators
}

// GetUpdatedValidators returns the current validators
func (strategy *Strategy) SetAccountMapList(accountMapList *tmTypes.AccountMapList) {
	strategy.AccountMapList = accountMapList
}