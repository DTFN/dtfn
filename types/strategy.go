package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/txfilter"
	"github.com/ethereum/go-ethereum/log"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	tmlibs "github.com/tendermint/tendermint/libs/common"
	"math/big"
	"fmt"
	"github.com/tendermint/tendermint/crypto"
	"github.com/green-element-chain/gelchain/version"
)

// MinerRewardStrategy is a mining strategy
type MinerRewardStrategy interface {
	Receiver() common.Address
}

// ValidatorsStrategy is a validator strategy
type ValidatorsStrategy interface {
	SetInitialValidators(validators []abciTypes.ValidatorUpdate)
	//CollectTx(tx *ethTypes.Transaction)
	GetUpdatedValidators(height int64, seed []byte) abciTypes.ResponseEndBlock
	Receiver() common.Address
	Signer() ethTypes.Signer
}

// Strategy encompasses all available strategies
type Strategy struct {
	ValidatorsStrategy

	InitialValidators []abciTypes.ValidatorUpdate

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

	AuthTable *txfilter.AuthTable

	// add for hard fork
	HFExpectedData HardForkExpectedData

	signer ethTypes.Signer
}

type NextEpochValData struct {
	// will be changed by addValidatorTx and removeValidatorTx.
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

	//This variable is used to record the block was generated by which version
	BlockVersion uint64
}

type CurrEpochValData struct {
	//deepcopy from NextEpochValData each epoch
	PosTable *txfilter.PosTable `json:"pos_table"`

	TotalBalance *big.Int `json:"total_balance"`
	MinorBonus   *big.Int `json:"-"` //all voted validators share this bonus per block.

	SelectCount int `json:"-"` //select count of each height
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

func (strategy *Strategy) GetUpdatedValidators(height int64, seed []byte) abciTypes.ResponseEndBlock {
	if height%txfilter.EpochBlocks != 0 {
		if seed != nil {
			//seed 存在的时，优先seed
			return strategy.enterSelectValidators(seed, -1)
		} else {
			//seed 不存在，选取height
			return strategy.enterSelectValidators(nil, height)
		}
	} else {
		return strategy.blsValidators(height)
	}
}

func (strategy *Strategy) enterSelectValidators(seed []byte, height int64) abciTypes.ResponseEndBlock {
	/*if app.strategy.BlsSelectStrategy {
	} else {
	}*/
	validatorsSlice := []abciTypes.ValidatorUpdate{}

	selectCount := strategy.CurrEpochValData.SelectCount //currently fixed
	poolLen := len(strategy.CurrEpochValData.PosTable.PosItemMap)
	if poolLen < 7 {
		fmt.Printf("PosTable.PosItemMap len < 7, current len %v \n", poolLen)
	}
	if selectCount == 0 { //0 means return full set each height
		selectCount = poolLen
	}

	// we use map to remember which validators selected has put into validatorSlice
	selectedValidators := make(map[string]int)

	if strategy.HFExpectedData.BlockVersion >= 2 {
		for i := 0; len(validatorsSlice) != selectCount; i++ {
			var tmPubKey crypto.PubKey
			var validator Validator
			var signer common.Address
			var pubKey abciTypes.PubKey
			var posItem txfilter.PosItem
			if height == -1 {
				//height=-1 表示 seed 存在，使用seed
				signer, posItem = strategy.CurrEpochValData.PosTable.SelectItemBySeedValue(seed, i)
			} else {
				//seed 不存在，使用height
				startIndex := height
				signer, posItem = strategy.CurrEpochValData.PosTable.SelectItemByHeightValue(startIndex + int64(i))
			}
			pubKey = posItem.PubKey
			tmPubKey, _ = tmTypes.PB2TM.PubKey(pubKey)
			tmAddress := tmPubKey.Address().String()
			if index, ok := selectedValidators[tmAddress]; ok {
				validatorsSlice[index].Power++
			} else {
				validatorUpdate := abciTypes.ValidatorUpdate{
					PubKey: pubKey,
					Power:  1000,
				}
				validator = Validator{
					validatorUpdate,
					signer,
				}
				//Remember tmPubKey.Address 's index in the currentValidators Array
				selectedValidators[tmAddress] = len(validatorsSlice)
				validatorsSlice = append(validatorsSlice, validatorUpdate)
				strategy.CurrentHeightValData.Validators[tmAddress] = validator
			}
		}
	} else {
		//select validators from posTable
		for i := 0; i < selectCount; i++ {
			var tmPubKey crypto.PubKey
			var validator Validator
			var signer common.Address
			var pubKey abciTypes.PubKey
			var posItem txfilter.PosItem
			if height == -1 {
				//height=-1 表示 seed 存在，使用seed
				signer, posItem = strategy.CurrEpochValData.PosTable.SelectItemBySeedValue(seed, i)
			} else {
				//seed 不存在，使用height
				startIndex := height
				signer, posItem = strategy.CurrEpochValData.PosTable.SelectItemByHeightValue(startIndex + int64(i))
			}
			pubKey = posItem.PubKey
			tmPubKey, _ = tmTypes.PB2TM.PubKey(pubKey)
			tmAddress := tmPubKey.Address().String()
			if index, ok := selectedValidators[tmAddress]; ok {
				validatorsSlice[index].Power++
			} else {
				validatorUpdate := abciTypes.ValidatorUpdate{
					PubKey: pubKey,
					Power:  1000,
				}
				validator = Validator{
					validatorUpdate,
					signer,
				}
				//Remember tmPubKey.Address 's index in the currentValidators Array
				selectedValidators[tmAddress] = len(validatorsSlice)
				validatorsSlice = append(validatorsSlice, validatorUpdate)
				strategy.CurrentHeightValData.Validators[tmAddress] = validator
			}
		}
	}

	//append the validators which will be deleted
	for address, v := range strategy.CurrentHeightValData.Validators {
		//tmPubKey, _ := tmTypes.PB2TM.PubKey(v.PubKey)
		index, selected := selectedValidators[address]
		if selected {
			v.Power = validatorsSlice[index].Power
		} else {
			validatorsSlice = append(validatorsSlice, abciTypes.ValidatorUpdate{
				PubKey: v.PubKey,
				Power:  0,
			})
			delete(strategy.CurrentHeightValData.Validators, address)
		}
	}

	abiEvent := strategy.getAuthTmItems(height)
	if abiEvent != nil {
		abiEvents := make([]abciTypes.Event, 0)
		abiEvents = append(abiEvents, *abiEvent)
		return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, Events: abiEvents, AppVersion: strategy.HFExpectedData.BlockVersion}
	}
	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, AppVersion: strategy.HFExpectedData.BlockVersion}
}

func (strategy *Strategy) blsValidators(height int64) abciTypes.ResponseEndBlock {
	blsPubkeySlice := []string{}
	validatorsSlice := []abciTypes.ValidatorUpdate{}
	topKSigners := strategy.CurrEpochValData.PosTable.TopKSigners(100)
	currentValidators := map[string]Validator{}

	for _, signer := range topKSigners {
		posItem := strategy.CurrEpochValData.PosTable.PosItemMap[signer]
		tmAddress := posItem.TmAddress
		updateValidator := abciTypes.ValidatorUpdate{
			PubKey: posItem.PubKey,
			Power:  posItem.Slots,
		}
		emtValidator := Validator{updateValidator, signer}
		currentValidators[tmAddress] = emtValidator
		validatorsSlice = append(validatorsSlice, updateValidator)
		blsPubkeySlice = append(blsPubkeySlice, posItem.BlsKeyString)
	}

	for tmAddress, v := range strategy.CurrentHeightValData.Validators {
		_, ok := currentValidators[tmAddress]
		if !ok {
			validatorsSlice = append(validatorsSlice,
				abciTypes.ValidatorUpdate{
					PubKey: v.PubKey,
					Power:  int64(0),
				})
		}
	}
	strategy.CurrentHeightValData.Validators = currentValidators

	abiEvents := make([]abciTypes.Event, 0)
	//get all validators and init tm-auth-table
	if height == version.HeightArray[3] {
		initEvent := abciTypes.Event{Type: "AuthTableInit"}
		abiEvents = append(abiEvents, initEvent)
		// Private PPChain Admin account
		txfilter.PPChainAdmin = common.HexToAddress(version.PPChainPrivateAdmin)
	}
	abiEvent := strategy.getAuthTmItems(height)
	if abiEvent != nil {
		abiEvents = append(abiEvents, *abiEvent)
	}
	if len(abiEvents) != 0 {
		return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, BlsKeyString: blsPubkeySlice, Events: abiEvents, AppVersion: strategy.HFExpectedData.BlockVersion}
	}
	return abciTypes.ResponseEndBlock{ValidatorUpdates: validatorsSlice, BlsKeyString: blsPubkeySlice, AppVersion: strategy.HFExpectedData.BlockVersion}
}

func (strategy *Strategy) getAuthTmItems(height int64) *abciTypes.Event {
	if strategy.HFExpectedData.BlockVersion >= 5 && len(strategy.AuthTable.ThisBlockChangedMap) != 0 {
		abiEvent := &abciTypes.Event{Type: "AuthItem"}
		for tmAddr, value := range strategy.AuthTable.ThisBlockChangedMap {
			var oper []byte
			if value {
				oper = append(oper, '1')

			}
			abiEvent.Attributes = append(abiEvent.Attributes, tmlibs.KVPair{
				Key:   []byte(tmAddr),
				Value: oper,
			})
		}

		//reset at the end of block
		strategy.AuthTable.ThisBlockChangedMap = make(map[string]bool)
		return abiEvent
	}

	//return an empty auth map on version<=4
	strategy.AuthTable.ThisBlockChangedMap = make(map[string]bool)
	return nil
}

// Receiver returns which address should receive the mining reward
func (strategy *Strategy) Receiver() common.Address {
	if strategy.HFExpectedData.BlockVersion == 4 {
		return txfilter.Bigguy //not good, all the coinbases in the headers are bigguy
	}
	if strategy.CurrentHeightValData.ProposerAddress == "" || len(strategy.CurrEpochValData.PosTable.TmAddressToSignerMap) == 0 {
		return common.HexToAddress("0000000000000000000000000000000000000002")
	} else if signer, ok := strategy.CurrEpochValData.PosTable.TmAddressToSignerMap[strategy.CurrentHeightValData.ProposerAddress]; ok {
		if pi, ok := strategy.CurrEpochValData.PosTable.PosItemMap[signer]; ok {
			return pi.Beneficiary
		}
		if pi, ok := strategy.CurrEpochValData.PosTable.UnbondPosItemMap[signer]; ok && pi.Slots != 0 { //pi.Slots==0 means it's a slashed account
			return pi.Beneficiary
		}
	}
	log.Error(fmt.Sprintf("Proposer Address %v not found in accountMap", strategy.CurrentHeightValData.ProposerAddress))
	return common.HexToAddress("0000000000000000000000000000000000000002")
}

// SetValidators the initial validators
func (strategy *Strategy) SetInitialValidators(validators []abciTypes.ValidatorUpdate) {
	strategy.InitialValidators = validators
}

func (strategy *Strategy) Signer() ethTypes.Signer {
	return strategy.signer
}

// CollectTx collects the rewards for a transaction
/*func (strategy *Strategy) CollectTx(tx *ethTypes.Transaction) {
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
}*/

func (strategy *Strategy) SetSigner(chainId *big.Int) {
	strategy.signer = ethTypes.NewEIP155Signer(chainId)
}

func (strategy *Strategy) SetInitialAccountMap(accountMapList *AccountMap) {
	//strategy.CurrHeightValData.AccountMap = accountMapList
	strategy.AccMapInitial = &AccountMap{
		MapList: accountMapList.MapList,
	}
}
