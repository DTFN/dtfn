package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"reflect"
	"sort"
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

	InitialValidators []abciTypes.ValidatorUpdate

	currentValidators []abciTypes.ValidatorUpdate //old code use. We don't use this
	//This map was used when some validator was removed when initial at initChain(i.e dont have enough money)
	// and didnt existed in the accountMapList
	// we should remember it for balance bonus and then clear it
	AccMapInitial *AccountMap

	//needn't to be persisted
	BlsSelectStrategy bool

	// reused,need persistence
	CurrHeightValData CurrentHeightValData

	//reused,need persistence
	NextEpochValData NextEpochValData

	// add for hard fork
	HFExpectedData HardForkExpectedData
}

type NextEpochValData struct {
	//we should deepcopy evert 200 height
	//first deepcopy:copy at height 1 from CurrentRoundValData to NextRoundValData
	//height/200 ==0:c from NextRoundValData to CurrentRoundValData   `json:"-"`
	NextPosTable *PosTable `json:"pos_table"`

	NextCandidateValidators map[string]abciTypes.ValidatorUpdate `json:"next_candidate_validators"`

	NextAccountMap *AccountMap `json:"account_map"`

	ChangedFlagThisBlock bool
	// whether upsert or remove in this block
	// when we need to write into the ethState ,we set it to true
}

func (nextRoundValData *NextEpochValData) ExportCandidateValidators() []abciTypes.ValidatorUpdate {
	var keys []string
	for k := range nextRoundValData.NextCandidateValidators {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	validators := []abciTypes.ValidatorUpdate{}
	for _, k := range keys {
		validators = append(validators, nextRoundValData.NextCandidateValidators[k])
	}
	return validators
}

type Proposer struct {
	Receiver string `json:"receiver"`
}

// no need to be persisted
type HardForkExpectedData struct {
	Height int64 // should remember and update it for every block to remember what height we located

	IsHarfForkPassed bool // This flag is used to record whether the hardfork was passed by most of validators

	// This flag is used to record the hard fork version that most of nodes want to upgrade
	// If the statistic process doesn't exist, statisticsVersion = 0 , use const NextHardForkVersion = 2
	StatisticsVersion uint64

	//This variable is used to record the statisticHeight that most of nodes want to upgrade
	//If the statistic process doesn't exist,statisticsHeight = 0 ,use const NextHardForkHeight = 2
	StatisticHeight int64

	//This variable is used to record the block was generated by whick version
	BlockVersion uint64
}

type CurrentHeightValData struct {
	Height int64

	AccountMap *AccountMap `json:"account_map"`
	//This map was used when some validator was removed and didnt existed in the accountMapList
	// we should remember it for balance bonus and then clear it
	//AccountMapListTemp *AccountMapList

	LastEpochAccountMap *AccountMap `json:"last_epoch_account_map"`

	// will be changed by addValidatorTx and removeValidatorTx.
	PosTable *PosTable `json:"pos_table"`

	// current candidate Validators , will changed every 200 height,will be changed by addValidatorTx and removeValidatorTx
	CurrCandidateValidators []abciTypes.ValidatorUpdate `json:"curr_candidate_validators"`

	// UpdateValidators of currentBlock, will use to set votePower to 0 ,then remove from tendermint validatorSet
	// will be select by posTable.
	// CurrentValidators is the true validators except committee validator when height != 1
	// if height =1 ,UpdateValidators = nil
	UpdateValidators []abciTypes.ValidatorUpdate //saved in another address separately

	TotalBalance *big.Int `json:"totalBalance"`
	MinorBonus   *big.Int //award all validators per block.

	ProposerAddress string

	LastVoteInfo []abciTypes.VoteInfo

	// note : if we get a addValidatorsTx at height 101,
	// we will put it into the NextCandidateValidators and move into postable
	// NextCandidateValidator will used in the next height200
	// postable will used in the next height 102

	//note : if we get a removeValidatorsTx at height 101
	// we will remove it from the NextCandidateValidators and remove from postable
	// NextCandidateValidator will used in the next height200
	// postable will used in the next height 102
}

type LastUpdateValidators struct {
	//used for persist data
	UpdateValidators []abciTypes.ValidatorUpdate `json:"update_validators"`
}

func NewStrategy(totalBalance *big.Int) *Strategy {
	//If ThresholdUnit = 1000 ,it mean we set the lowest posTable threshold to 1/1000 of totalBalance.
	thresholdUnit := big.NewInt(ThresholdUnit)
	threshold := big.NewInt(1)
	hfExpectedData := HardForkExpectedData{Height: 0, IsHarfForkPassed: true, StatisticsVersion: 0, BlockVersion: 0}
	return &Strategy{
		CurrHeightValData: CurrentHeightValData{
			PosTable:            NewPosTable(threshold.Div(totalBalance, thresholdUnit)),
			AccountMap:          &AccountMap{MapList: map[string]*AccountMapItem{}},
			LastEpochAccountMap: &AccountMap{MapList: map[string]*AccountMapItem{}},
			TotalBalance:        totalBalance,
		},
		HFExpectedData: hfExpectedData,

		NextEpochValData: NextEpochValData{
			NextPosTable: NewPosTable(threshold.Div(totalBalance, thresholdUnit)),
			NextAccountMap: &AccountMap{
				MapList: make(map[string]*AccountMapItem),
			},
			NextCandidateValidators: map[string]abciTypes.ValidatorUpdate{},
		},
	}
}

// Receiver returns which address should receive the mining reward
func (s *Strategy) Receiver() common.Address {
	if s.CurrHeightValData.ProposerAddress == "" || len(s.CurrHeightValData.AccountMap.MapList) == 0 {
		return common.HexToAddress("0000000000000000000000000000000000000002")
	} else if s.CurrHeightValData.AccountMap.MapList[s.CurrHeightValData.ProposerAddress] != nil {
		return s.CurrHeightValData.AccountMap.MapList[s.CurrHeightValData.ProposerAddress].Beneficiary
	}
	log.Error("Proposer Address %v not found in accountMap", s.CurrHeightValData.ProposerAddress)
	return common.HexToAddress("0000000000000000000000000000000000000002")
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

// GetUpdatedValidators returns the current validators,  old code
func (strategy *Strategy) GetUpdatedValidators() []abciTypes.ValidatorUpdate {
	return strategy.currentValidators
}

func (strategy *Strategy) SetInitialAccountMap(accountMapList *AccountMap) {
	//strategy.CurrHeightValData.AccountMap = accountMapList
	strategy.AccMapInitial = &AccountMap{
		MapList: accountMapList.MapList,
	}
}
