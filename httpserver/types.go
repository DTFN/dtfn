package httpserver

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/green-element-chain/gelchain/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

type Validator struct {
	Address       []byte           `json:"address"`
	PubKey        abciTypes.PubKey `json:"pubKey"`
	Power         int64            `json:"power"`
	AddressString string           `json:"addressString"`
}

type Pos struct {
	NextCandidateValidators []*Validator `json:"nextValidators"`

	AccountMapList *tmTypes.AccountMapList `json:"accountMap"`

	PosItemMap map[common.Address]*types.PosItem `json:"posTableMap"`

	Success bool `json:"success"`
}
