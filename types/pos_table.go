package types

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"sync"
)

type PosTable struct {
	mtx          sync.RWMutex
	PosItemMap   map[common.Address]*PosItem `json:"accounts"` //This isnt called by foreign struct except rpc
	PosArray     []*PosItem                  // All posItem
	PosArraySize int                         // real size of posArray
	threshold    *big.Int                    // threshold value of PosTable
}

func NewPosTable(threshold *big.Int) *PosTable {
	pa := make([]*PosItem, 2000)
	return &PosTable{
		PosItemMap:   make(map[common.Address]*PosItem),
		PosArray:     pa,
		PosArraySize: 0,
		threshold:    threshold,
	}
}

func (posTable *PosTable) UpsertPosItem(signer common.Address, balance *big.Int, beneficiary common.Address,
	pubkey abciTypes.PubKey) (bool, error) {
	posTable.mtx.Lock()
	defer posTable.mtx.Unlock()

	balanceCopy := big.NewInt(1000)

	posOriginPtr, exist := posTable.PosItemMap[signer]
	if exist {
		posTableCopy := big.NewInt(1000)
		originPosWeight, _ := strconv.Atoi(posTableCopy.
			Div(posTable.PosItemMap[signer].Balance, posTable.threshold).String())
		newPosWeight, _ := strconv.Atoi(balanceCopy.Div(balance, posTable.threshold).String())
		if originPosWeight >= newPosWeight {
			return false, fmt.Errorf("situation should happened in real world")
		} else {
			for i := 0; i < newPosWeight-originPosWeight; i++ {
				posTable.PosArray[posTable.PosArraySize] = posOriginPtr
				posOriginPtr.Indexes[posTable.PosArraySize] = true
				posTable.PosArraySize++
			}
			posTable.PosItemMap[signer].Balance = balance
			return true, nil
		}
	}
	if balance.Cmp(posTable.threshold) < 0 {
		return false, fmt.Errorf("balance not enought")
	}
	posItemPtr := newPosItem(signer, balance, beneficiary, pubkey)
	posTable.PosItemMap[signer] = posItemPtr
	posNumber, _ := strconv.Atoi(balanceCopy.Div(balance, posTable.threshold).String())
	for i := 0; i < posNumber; i++ {
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

func (posTable *PosTable) SetThreShold(threShold *big.Int) {
	posTable.threshold = threShold
}

func (posTable *PosTable) SelectItemByHeightValue(random int) PosItem {
	r := rand.New(rand.NewSource(int64(random)))
	return *posTable.PosArray[r.Intn(posTable.PosArraySize)]
}

func (posTable *PosTable) SelectItemBySeedValue(vrf []byte, i int) PosItem {

	var v1, v2 uint32
	k := uint32(i)
	bIdx := k / 8
	bits1 := k % 8
	bits2 := 8 + bits1 // L - 8 + bits1

	v1 = uint32(vrf[bIdx]) >> bits1
	if bIdx+1 < uint32(len(vrf)) {
		v2 = uint32(vrf[bIdx+1])
	} else {
		v2 = uint32(vrf[0])
	}

	v2 = v2 & ((1 << bits2) - 1)
	v := (v2 << (8 - bits1)) + v1
	v = v % uint32(len(posTable.PosArray))
	return *posTable.PosArray[v]
}

type PosItem struct {
	Signer      common.Address
	Balance     *big.Int
	PubKey      abciTypes.PubKey
	Indexes     map[int]bool
	Beneficiary common.Address
}

func newPosItem(signer common.Address, balance *big.Int, beneficiary common.Address, pubKey abciTypes.PubKey) *PosItem {
	return &PosItem{
		Signer:      signer,
		Balance:     balance,
		PubKey:      pubKey,
		Indexes:     make(map[int]bool),
		Beneficiary: beneficiary,
	}
}
