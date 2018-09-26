package types

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/crypto"
	tmTypes "github.com/tendermint/tendermint/types"
	"sort"
	"sync"
)

type PosTable struct {
	mtx          sync.RWMutex
	posItemMap   map[common.Address]*posItem `json:"accounts"`
	posArray     []*posItem                  // All posItem
	posArraySize int                         // real size of posArray
	threshold    int64                       // threshold value of PosTable
}

func NewPosTable(threshold int64) *PosTable {
	pa := make([]*posItem, 2000)
	return &PosTable{
		posItemMap:   make(map[common.Address]*posItem),
		posArray:     pa,
		posArraySize: 0,
		threshold:    threshold,
	}
}

func (posTable *PosTable) UpsertPosItem(account common.Address, balance int64, address tmTypes.Address,
	pubkey crypto.PubKey) (bool, error) {
	posTable.mtx.Lock()
	defer posTable.mtx.Unlock()

	posOriginPtr, exist := posTable.posItemMap[account]
	if exist {
		originPos := int(posTable.posItemMap[account].Balance / posTable.threshold)
		newPos := int(balance / posTable.threshold)
		if originPos >= newPos {
			return false, fmt.Errorf("address not upsert")
		} else {
			for i := 0; i < newPos-originPos; i++ {
				posTable.posArray[posTable.posArraySize] = posOriginPtr
				posOriginPtr.Indexes[posTable.posArraySize] = true
				posTable.posArraySize++
			}
			posTable.posItemMap[account].Balance = balance
			return true, nil
		}
	}
	if balance < posTable.threshold {
		return false, fmt.Errorf("balance not enought")
	}
	posItemPtr := newPosItem(account, balance, address, pubkey)
	posTable.posItemMap[account] = posItemPtr
	for i := 0; i < int(balance/posTable.threshold); i++ {
		posTable.posArray[posTable.posArraySize] = posItemPtr
		posItemPtr.Indexes[posTable.posArraySize] = true
		posTable.posArraySize++
	}
	return true, nil
}

func (posTable *PosTable) RemovePosItem(account common.Address) (bool, error) {
	posTable.mtx.Lock()
	defer posTable.mtx.Unlock()

	_, exist := posTable.posItemMap[account]
	if !exist {
		return false, fmt.Errorf("address not existed in the postable")
	}

	posItemPtr := posTable.posItemMap[account]
	var indexArray []int
	for k, _ := range posItemPtr.Indexes {
		indexArray = append(indexArray, k)
	}
	sort.Ints(indexArray)

	for i := len(indexArray) - 1; i >= 0; i-- {
		if indexArray[i] == posTable.posArraySize-1 {
			posTable.posArraySize--
		} else {
			newPosItem := posTable.posArray[posTable.posArraySize-1]
			posTable.posArray[indexArray[i]] = newPosItem
			newPosItem.Indexes[indexArray[i]] = true

			delete(newPosItem.Indexes, posTable.posArraySize-1)
			posTable.posArraySize--
		}
	}

	delete(posTable.posItemMap, account)

	return true, nil
}

func (posTable *PosTable) SetThreShold(threShold int64) {
	posTable.threshold = threShold
}

func (posTable *PosTable) SelectItemByRandomValue(random int) posItem {
	return *posTable.posItemMap[common.HexToAddress("0000000000000000000000000000000000000001")]
}

type posItem struct {
	Account common.Address
	Balance int64
	PubKey  crypto.PubKey
	Indexes map[int]bool
	Address tmTypes.Address
}

func newPosItem(account common.Address, balance int64, address tmTypes.Address, pubKey crypto.PubKey) *posItem {
	return &posItem{
		Account: account,
		Balance: balance,
		PubKey:  pubKey,
		Indexes: make(map[int]bool),
		Address: address,
	}
}
