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
	currentValidators []*abciTypes.Validator
	AccountMapList    *tmTypes.AccountMapList

	//This map was used when some validator was removed and didnt existed in the accountMapList
	// we should remember it for balance bonus and then clear it
	AccountMapListTemp *tmTypes.AccountMapList

	FirstInitial bool

	ProposerAddress string

	ValidatorSet Validators

	// will be changed by addValidatorTx and removeValidatorTx.
	PosTable *PosTable

	TotalBalance *big.Int
}

type Validators struct {

	// Next candidate Validators , will changed every 200 height,will be changed by addValidatorTx and removeValidatorTx
	NextHeightCandidateValidators []*abciTypes.Validator

	// Initial validators , only use for once
	InitialValidators []*abciTypes.Validator

	// validators of currentBlock, will use to set votePower to 0 ,then remove from tendermint validatorSet
	// will be select by postable.
	// CurrentValidators is the true validators except commmittee validator when height != 1
	// if height =1 ,CurrentValidator = nil
	CurrentValidators []*abciTypes.Validator

	// current validator weight represent the weight of random select.
	// will used to accumulateReward for next height
	CurrentValidatorWeight []int64

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
	threshold := big.NewInt(1000)
	return &Strategy{
		PosTable:     NewPosTable(threshold.Div(totalBalance, threshold)),
		TotalBalance: totalBalance,
	}
}

// Receiver returns which address should receive the mining reward
func (s *Strategy) Receiver() common.Address {
	if s.ProposerAddress == "" || len(s.AccountMapList.MapList) == 0 {
		return common.HexToAddress("0000000000000000000000000000000000000002")
	} else if s.AccountMapList.MapList[s.ProposerAddress] != nil {
		return s.AccountMapList.MapList[s.ProposerAddress].Beneficiary
	} else {
		return s.AccountMapListTemp.MapList[s.ProposerAddress].Beneficiary
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
	strategy.AccountMapListTemp = &tmTypes.AccountMapList{
		MapList: make(map[string]*tmTypes.AccountMap),
	}
}
