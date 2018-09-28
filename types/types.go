package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
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

	//if height = 1 ,currentValidator come from genesis.json
	//if height != 1, currentValidator == Validators.CurrentValidators + committeeValidators
	currentValidators  []*abciTypes.Validator
	AccountMapList     *tmTypes.AccountMapList
	ValidatorTmAddress string

	ValidatorSet Validators

	// will be changed by addValidatorTx and removeValidatorTx.
	PosTable *PosTable

	TotalBalance *big.Int
}

type Validators struct {
	// validators of committee , used to support +2/3 ,our node
	CommitteeValidators []*abciTypes.Validator

	// current validators of candidate
	CandidateValidators []*abciTypes.Validator

	// Next candidate Validators , will changed every 200 height,will be changed by addValidatorTx and removeValidatorTx
	NextCandidateValidators []*abciTypes.Validator

	// validators of currentBlock, will use to set votePower to 0 ,then remove from tendermint validatorSet
	// will be select by postable.
	// CurrentValidators is the true validators except commmittee validator when height != 1
	// if height =1 ,CurrentValidator = nil
	CurrentValidators []*abciTypes.Validator

	// note : if we get a addValidatorsTx at height 101,
	// we will put it into the NextCandidateValidators and move into postable
	// NextCandidateValidator will used in the next height200
	// postable will used in the next height 102

	//note : if we get a removeValidatorsTx at height 101
	// we will remove it from the NextCandidateValidators and remove from postable
	// NextCandidateValidator will used in the next height200
	// postable will used in the next height 102
}

func NewStrategy(totalBalance *big.Int) *Strategy {
	return &Strategy{
		PosTable:     NewPosTable(int64(1000)),
		TotalBalance: totalBalance,
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
