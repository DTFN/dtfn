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
	mtx                  sync.RWMutex
	PosItemMap           map[common.Address]*PosItem `json:"posItemMap"`   //This isnt called by foreign struct except rpc
	PosArray             []common.Address            `json:"posArray"`     // All posItem
	PosArraySize         int                         `json:"posArraySize"` // real size of posArray
	Threshold            *big.Int                    `json:"threshold"`    // threshold value of PosTable
	ChangedFlagThisBlock bool                        `json:"changedFlagThisBlock"`
	// whether upsert or remove in this block
	// but when we need to write into the ethstate ,we set it to false
}

func NewPosTable(threshold *big.Int) *PosTable {
	pa := make([]common.Address, 2000)
	return &PosTable{
		PosItemMap:   make(map[common.Address]*PosItem),
		PosArray:     pa,
		PosArraySize: 0,
		Threshold:    threshold,
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
			Div(posTable.PosItemMap[signer].Balance, posTable.Threshold).String())
		newPosWeight, _ := strconv.Atoi(balanceCopy.Div(balance, posTable.Threshold).String())
		if originPosWeight >= newPosWeight {
			return false, fmt.Errorf("situation should happened in real world")
		} else {
			for i := 0; i < newPosWeight-originPosWeight; i++ {
				posTable.PosArray[posTable.PosArraySize] = signer
				posOriginPtr.Indexes[posTable.PosArraySize] = true
				posTable.PosArraySize++
			}
			posTable.PosItemMap[signer].Balance = balance
			return true, nil
		}
	}
	if balance.Cmp(posTable.Threshold) < 0 {
		return false, fmt.Errorf("balance not enought")
	}
	posItemPtr := newPosItem(signer, balance, beneficiary, pubkey)
	posTable.PosItemMap[signer] = posItemPtr
	posNumber, _ := strconv.Atoi(balanceCopy.Div(balance, posTable.Threshold).String())
	for i := 0; i < posNumber; i++ {
		posTable.PosArray[posTable.PosArraySize] = signer
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
			newPosItem := posTable.PosItemMap[posTable.PosArray[posTable.PosArraySize-1]]
			posTable.PosArray[indexArray[i]] = posTable.PosArray[posTable.PosArraySize-1]
			newPosItem.Indexes[indexArray[i]] = true

			delete(newPosItem.Indexes, posTable.PosArraySize-1)
			posTable.PosArraySize--
		}
	}

	delete(posTable.PosItemMap, account)

	return true, nil
}

func (posTable *PosTable) SetThreShold(threShold *big.Int) {
	posTable.Threshold = threShold
}

func (posTable *PosTable) SelectItemByHeightValue(random int) PosItem {
	r := rand.New(rand.NewSource(int64(random)))
	return *posTable.PosItemMap[posTable.PosArray[r.Intn(posTable.PosArraySize)]]
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
	v = v % uint32(posTable.PosArraySize) //len(posTable.PosArray)
	return *posTable.PosItemMap[posTable.PosArray[v]]
}

type PosItem struct {
	Signer      common.Address   `json:"signer"`
	Balance     *big.Int         `json:"balance"`
	PubKey      abciTypes.PubKey `json:"pubKey"`
	Indexes     map[int]bool     `json:"indexes"`
	Beneficiary common.Address   `json:"beneficiary"`
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
