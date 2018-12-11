package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	abciTypes "github.com/tendermint/tendermint/abci/types"
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
	//only need once,dont need persistence
	//needn't to be persisted
	currentValidators []abciTypes.ValidatorUpdate

	//need to be persisted
	FirstInitial bool

	//needn't to be persisted
	BlsSelectStrategy bool

	// reused,need persistence
	CurrRoundValData CurrentRoundValData

	//reused,need persistence
	NextRoundValData NextRoundValData

	// add for hard fork
	HFExpectedData HardForkExpectedData
}

type NextRoundValData struct {
	//we should deepcopy evert 200 height
	//first deepcopy:copy at height 1 from CurrentRoundValData to NextRoundValData
	//height/200 ==0:c from NextRoundValData to CurrentRoundValData   `json:"-"`
	NextRoundPosTable *PosTable `json:"nextRoundPosTable"`

	NextRoundCandidateValidators []abciTypes.ValidatorUpdate `json:"nextRoundCandidateValidators"`

	NextAccountMapList *AccountMapList `json:"nextAccountMapList"`
}

type Proposer struct {
	Receiver string `json:"receiver"`
}

// no need to be persisted
type HardForkExpectedData struct {
	Height int64 // should remember and update it for every block to remember what height we located

	IsHarfForkPassed bool // This flag is used to record whether the hardfork was passed by most of validators

	// This flag is remember the hard fork version that most of nodes want to upgrade
	// If the statistic process doesn't exist, statisticsVersion = 0
	StatisticsVersion int

}

type CurrentRoundValData struct {
	AccountMapList *AccountMapList `json:"accountMapList"`
	//This map was used when some validator was removed and didnt existed in the accountMapList
	// we should remember it for balance bonus and then clear it
	//AccountMapListTemp *AccountMapList

	//This map was used when some validator was removed when initial at initChain(i.e dont have enough money)
	// and didnt existed in the accountMapList
	// we should remember it for balance bonus and then clear it
	AccMapInitial *AccountMapList `json:"accMapInitial"`

	// will be changed by addValidatorTx and removeValidatorTx.
	PosTable *PosTable `json:"posTable"`

	// current candidate Validators , will changed every 200 height,will be changed by addValidatorTx and removeValidatorTx
	CurrCandidateValidators []abciTypes.ValidatorUpdate `json:"currCandidateValidators"`

	// Initial validators , only use for once
	InitialValidators []abciTypes.ValidatorUpdate `json:"initialValidators"`

	// validators of currentBlock, will use to set votePower to 0 ,then remove from tendermint validatorSet
	// will be select by postable.
	// CurrentValidators is the true validators except commmittee validator when height != 1
	// if height =1 ,CurrentValidator = nil
	CurrentValidators []abciTypes.ValidatorUpdate `json:"currentValidators"`

	// current validator weight represent the weight of random select.
	// will used to accumulateReward for next height
	CurrentValidatorWeight []int64 `json:"currentValidatorWeight"`

	TotalBalance *big.Int `json:"totalBalance"`

	ProposerAddress string `json:"proposerAddress"`

	Receiver string `json:"receiver"`

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
	//If ThresholdUnit = 1000 ,it mean we set the lowest posTable threshold to 1/1000 of totalBalance.
	thresholdUnit := big.NewInt(ThresholdUnit)
	threshold := big.NewInt(1)
	hfExpectedData := HardForkExpectedData{Height: 0, IsHarfForkPassed: true,StatisticsVersion:0}
	return &Strategy{
		CurrRoundValData: CurrentRoundValData{
			PosTable:     NewPosTable(threshold.Div(totalBalance, thresholdUnit)),
			TotalBalance: totalBalance,
		},
		HFExpectedData: hfExpectedData,

		NextRoundValData: NextRoundValData{
			NextRoundPosTable: NewPosTable(threshold.Div(totalBalance, thresholdUnit)),
			NextAccountMapList: &AccountMapList{
				MapList: make(map[string]*AccountMap),
			},
		},
	}
}

// Receiver returns which address should receive the mining reward
func (s *Strategy) Receiver() common.Address {
	if s.CurrRoundValData.ProposerAddress == "" || len(s.CurrRoundValData.AccountMapList.MapList) == 0 {
		return common.HexToAddress("0000000000000000000000000000000000000002")
	} else if s.CurrRoundValData.AccountMapList.MapList[s.CurrRoundValData.ProposerAddress] != nil {
		return s.CurrRoundValData.AccountMapList.MapList[s.CurrRoundValData.ProposerAddress].Beneficiary
	} else {
		return s.CurrRoundValData.AccMapInitial.MapList[s.CurrRoundValData.ProposerAddress].Beneficiary
	}
}

// SetValidators updates the current validators
func (strategy *Strategy) SetValidators(validators []abciTypes.ValidatorUpdate) {
	strategy.currentValidators = validators
}

// CollectTx collects the rewards for a transaction
func (strategy *Strategy) CollectTx(tx *ethTypes.Transaction) {
	if reflect.DeepEqual(tx.To(), common.HexToAddress("0000000000000000000000000000000000000001")) {
		log.Info("Adding validator", "data", tx.Data())
		pubKey := abciTypes.PubKey{Data: tx.Data()}
		strategy.currentValidators = append(
			strategy.currentValidators,
			abciTypes.ValidatorUpdate{
				PubKey: pubKey,
				Power:  tx.Value().Int64(),
			},
		)
	}
}

// GetUpdatedValidators returns the current validators
func (strategy *Strategy) GetUpdatedValidators() []abciTypes.ValidatorUpdate {
	return strategy.currentValidators
}

// GetUpdatedValidators returns the current validators
func (strategy *Strategy) SetAccountMapList(accountMapList *AccountMapList) {
	strategy.CurrRoundValData.AccountMapList = accountMapList
	strategy.CurrRoundValData.AccMapInitial = &AccountMapList{
		MapList: make(map[string]*AccountMap),
	}
}
