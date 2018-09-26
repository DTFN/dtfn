package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"reflect"
)

// MinerRewardStrategy is a mining strategy
type MinerRewardStrategy interface {
	Receiver() common.Address
}

// ValidatorsStrategy is a validator strategy
type ValidatorsStrategy interface {
	SetValidators(validators []*abciTypes.Validator)
	CollectTx(tx *ethTypes.Transaction)
	GetUpdatedValidators() []*abciTypes.Validator
}

// Strategy encompasses all available strategies
type Strategy struct {
	MinerRewardStrategy
	ValidatorsStrategy

	currentValidators  []*abciTypes.Validator
	AccountMapList     *tmTypes.AccountMapList
	ValidatorTmAddress string

	ValidatorSet Validators
	PosTable     *PosTable
}

type Validators struct {
	// validators of committee , used to support +2/3
	CommitteeValidators []*abciTypes.Validator

	// validators of candidate ,will be changed by addValidatorTx and removeValidatorTx
	CandidateValidators []*abciTypes.Validator

	// validators of currentBlock, will use to set votePower to 0 ,then remove from tendermint validatorSet
	CurrentValidators []*abciTypes.Validator
}

func NewStrategy() *Strategy {
	return &Strategy{
		PosTable: NewPosTable(int64(1000)),
	}
}

// Receiver returns which address should receive the mining reward
func (s *Strategy) Receiver() common.Address {
	if s.ValidatorTmAddress == "" {
		return common.HexToAddress("0000000000000000000000000000000000000002")
	} else {
		return s.AccountMapList.MapList[s.ValidatorTmAddress].Beneficiary
	}
}

// SetValidators updates the current validators
func (strategy *Strategy) SetValidators(validators []*abciTypes.Validator) {
	strategy.currentValidators = validators
}

// CollectTx collects the rewards for a transaction
func (strategy *Strategy) CollectTx(tx *ethTypes.Transaction) {
	if reflect.DeepEqual(tx.To(), common.HexToAddress("0000000000000000000000000000000000000001")) {
		log.Info("Adding validator", "data", tx.Data())
		pubKey := abciTypes.PubKey{Data: tx.Data()}
		strategy.currentValidators = append(
			strategy.currentValidators,
			&abciTypes.Validator{
				PubKey: pubKey,
				Power:  tx.Value().Int64(),
			},
		)
	}
}

// GetUpdatedValidators returns the current validators
func (strategy *Strategy) GetUpdatedValidators() []*abciTypes.Validator {
	return strategy.currentValidators
}

// GetUpdatedValidators returns the current validators
func (strategy *Strategy) SetAccountMapList(accountMapList *tmTypes.AccountMapList) {
	strategy.AccountMapList = accountMapList
}
