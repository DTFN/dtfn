package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/txfilter"
	"github.com/ethereum/go-ethereum/log"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"reflect"
	"fmt"
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

	//initial bond accounts
	AccMapInitial *AccountMap

	//needn't to be persisted
	BlsSelectStrategy bool

	// need persist every height
	CurrentHeightValData CurrentHeightValData

	// need persist when epoch changes
	CurrEpochValData CurrEpochValData

	// need persist when epoch changes or changed this block
	NextEpochValData NextEpochValData

	// add for hard fork
	HFExpectedData HardForkExpectedData
}

type NextEpochValData struct {
	//deepcopy from NextEpochValData to CurrEpochValData each epoch
	PosTable *txfilter.PosTable
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

type CurrEpochValData struct {
	// will be changed by addValidatorTx and removeValidatorTx.
	PosTable *txfilter.PosTable `json:"pos_table"`

	TotalBalance *big.Int `json:"total_balance"`
	MinorBonus   *big.Int `json:"-"` //all voted validators share this bonus per block.
}

type Validator struct {
	abciTypes.ValidatorUpdate
	Signer common.Address
}

type CurrentHeightValData struct {
	Height int64 `json:"height"`

	Validators map[string]Validator `json:"validators"`

	ProposerAddress string `json:"-"`

	LastVoteInfo []abciTypes.VoteInfo `json:"-"`
}

func NewStrategy() *Strategy {
	//If ThresholdUnit = 1000 ,it mean we set the lowest posTable threshold to 1/1000 of totalBalance.
	//thresholdUnit := big.NewInt(txfilter.ThresholdUnit)
	//threshold := big.NewInt(1)
	hfExpectedData := HardForkExpectedData{Height: 0, IsHarfForkPassed: true, StatisticsVersion: 0, BlockVersion: 0}
	return &Strategy{
		CurrEpochValData: CurrEpochValData{
			PosTable: nil, //later assigned in InitPersistData
		},
		HFExpectedData: hfExpectedData,

		NextEpochValData: NextEpochValData{
			PosTable: nil, //later assigned in InitPersistData
		},
		CurrentHeightValData: CurrentHeightValData{
			Height:     0,
			Validators: make(map[string]Validator),
		},
	}
}

// Receiver returns which address should receive the mining reward
func (s *Strategy) Receiver() common.Address {
	if s.CurrentHeightValData.ProposerAddress == "" || len(s.CurrEpochValData.PosTable.TmAddressToSignerMap) == 0 {
		return common.HexToAddress("0000000000000000000000000000000000000002")
	} else if signer, ok := s.CurrEpochValData.PosTable.TmAddressToSignerMap[s.CurrentHeightValData.ProposerAddress]; ok {
		if pi, ok := s.CurrEpochValData.PosTable.PosItemMap[signer]; ok {
			return pi.Beneficiary
		}
		if pi, ok := s.CurrEpochValData.PosTable.UnbondPosItemMap[signer]; ok && pi.Slots != 0 { //pi.Slots==0 means it's a slashed account
			return pi.Beneficiary
		}
	}
	log.Error(fmt.Sprintf("Proposer Address %v not found in accountMap", s.CurrentHeightValData.ProposerAddress))
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
