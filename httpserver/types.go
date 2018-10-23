package httpserver

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/ethermint/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

type Validator struct {
	Address       []byte           `json:"address"`
	PubKey        abciTypes.PubKey `json:"pubKey"`
	Power         int64            `json:"power"`
	AddressString string           `json:"addressString"`
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
