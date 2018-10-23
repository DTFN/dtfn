package httpserver

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/ethermint/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
)

type Validator struct {
	Address       []byte           `json:"address"`
	PubKey        abciTypes.PubKey `json:"pubKey"`
	Power         int64            `json:"power"`
	AddressString string           `json:"addressString"`
	Signer        common.Address   `json:"signer"`
	SignerBalance *big.Int         `json:"signerBalance"`
	Beneficiary   common.Address   `json:"beneficiary"`
}

type PTableAll struct {
	NextCandidateValidators []*Validator `json:"nextValidators"`

	AccountMapList *tmTypes.AccountMapList `json:"accountMap"`

	PosItemMap map[common.Address]*types.PosItem `json:"posTableMap"`

	Success bool `json:"success"`
}

type AccountMapData struct {
	MapList map[string]*tmTypes.AccountMap `json:"accountmaplist"`
}

type PosItemMapData struct {
	PosItemMap map[common.Address]*types.PosItem `json:"posTableMap"`
}

type PreBlockProposer struct {
	PreBlockProposer string         `json:"proposer"`
	Beneficiary      common.Address `json:"beneficiary"`
	Signer           common.Address `json:"signer"`
}

type PreBlockValidatorElect struct {
	preBlockValidators []*Validator `json:"preBlockValidators"`
}
