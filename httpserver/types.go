package httpserver

import (
	"github.com/ethereum/go-ethereum/common"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"github.com/ethereum/go-ethereum/core/txfilter"
)

type Validator struct {
	//Address       []byte           `json:"address"`
	PubKey        abciTypes.PubKey `json:"pubKey"`
	Power         int64            `json:"power"`
	AddressString string           `json:"addressString"`
	Signer        common.Address   `json:"signer"`
	SignerBalance *big.Int         `json:"signerBalance"`
	Beneficiary   common.Address   `json:"beneficiary"`
	BlsKeyString  string           `json:"blsKeyString"`
}

type PTableAll struct {
	NextCandidateValidators []*Validator `json:"next_validators"`

	AccountMapList map[string]common.Address `json:"account_map"`

	PosItemMap map[common.Address]*txfilter.PosItem `json:"pos_table_map"`

	Success bool `json:"success"`
}

type AccountMapData struct {
	MapList map[string]common.Address `json:"map_list"`
}

type PosItemMapData struct {
	PosItemMap   map[common.Address]*txfilter.PosItem `json:"pos_table_map"`
	Threshold    *big.Int                          `json:"threshold"`
	TotalSlots int64                               `json:"total_slots"`
}

type PreBlockProposer struct {
	PreBlockProposer string         `json:"proposer"`
	Beneficiary      common.Address `json:"beneficiary"`
	Signer           common.Address `json:"signer"`
}

type PreBlockValidatorElect struct {
	PreBlockValidators []*Validator `json:"preBlockValidators"`
}

type Encourage struct {
	TotalBalance          *big.Int `json:"initialTotalBalance"`
	EncourageAverageBlock *big.Int `json:"encourageAverageBlock"`
}
