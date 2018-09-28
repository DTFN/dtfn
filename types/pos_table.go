package types

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"sort"
	"sync"
)

type PosTable struct {
	mtx          sync.RWMutex
	PosItemMap   map[common.Address]*posItem `json:"accounts"`
	PosArray     []*posItem                  // All posItem
	PosArraySize int                         // real size of posArray
	threshold    int64                       // threshold value of PosTable
	PrimeArray   *PrimeArray
}

func NewPosTable(threshold int64) *PosTable {
	pa := make([]*posItem, 2000)
	return &PosTable{
		PosItemMap:   make(map[common.Address]*posItem),
		PosArray:     pa,
		PosArraySize: 0,
		threshold:    threshold,
		PrimeArray:   NewPrimeArray(),
	}
}

func (posTable *PosTable) UpsertPosItem(signer common.Address, balance int64, beneficiary common.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	posTable.mtx.Lock()
	defer posTable.mtx.Unlock()

	posOriginPtr, exist := posTable.PosItemMap[signer]
	if exist {
		originPos := int(posTable.PosItemMap[signer].Balance / posTable.threshold)
		newPos := int(balance / posTable.threshold)
		if originPos >= newPos {
			return false, fmt.Errorf("address not upsert")
		} else {
			for i := 0; i < newPos-originPos; i++ {
				posTable.PosArray[posTable.PosArraySize] = posOriginPtr
				posOriginPtr.Indexes[posTable.PosArraySize] = true
				posTable.PosArraySize++
			}
			posTable.PosItemMap[signer].Balance = balance
			return true, nil
		}
	}
	if balance < posTable.threshold {
		return false, fmt.Errorf("balance not enought")
	}
	posItemPtr := newPosItem(signer, balance, beneficiary, pubkey)
	posTable.PosItemMap[signer] = posItemPtr
	for i := 0; i < int(balance/posTable.threshold); i++ {
		posTable.PosArray[posTable.PosArraySize] = posItemPtr
		posItemPtr.Indexes[posTable.PosArraySize] = true
		posTable.PosArraySize++
	}
	return true, nil
}

func (posTable *PosTable) RemovePosItem(account common.Address) (bool, error) {
	posTable.mtx.Lock()
	defer posTable.mtx.Unlock()

	_, exist := posTable.PosItemMap[account]
	if !exist {
		return false, fmt.Errorf("address not existed in the postable")
	}

	posItemPtr := posTable.PosItemMap[account]
	var indexArray []int
	for k, _ := range posItemPtr.Indexes {
		indexArray = append(indexArray, k)
	}
	sort.Ints(indexArray)

	for i := len(indexArray) - 1; i >= 0; i-- {
		if indexArray[i] == posTable.PosArraySize-1 {
			posTable.PosArraySize--
		} else {
			newPosItem := posTable.PosArray[posTable.PosArraySize-1]
			posTable.PosArray[indexArray[i]] = newPosItem
			newPosItem.Indexes[indexArray[i]] = true

			delete(newPosItem.Indexes, posTable.PosArraySize-1)
			posTable.PosArraySize--
		}
	}

	delete(posTable.PosItemMap, account)

	return true, nil
}

func (posTable *PosTable) SetThreShold(threShold int64) {
	posTable.threshold = threShold
}

func (posTable *PosTable) SelectItemByRandomValue(random int) posItem {
	return *posTable.PosArray[random]
	//return *posTable.PosItemMap[common.HexToAddress("0000000000000000000000000000000000000001")]
}

type posItem struct {
	Signer      common.Address
	Balance     int64
	PubKey      abciTypes.PubKey
	Indexes     map[int]bool
	Beneficiary common.Address
}

func newPosItem(signer common.Address, balance int64, beneficiary common.Address, pubKey abciTypes.PubKey) *posItem {
	return &posItem{
		Signer:      signer,
		Balance:     balance,
		PubKey:      pubKey,
		Indexes:     make(map[int]bool),
		Beneficiary: beneficiary,
	}
}
