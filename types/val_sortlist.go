package types

import (
	"bytes"
	"container/list"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type ValSortlist struct {
	ValList *list.List
	Len     int
}

type ValListItem struct {
	Signer     common.Address `json:"signer"`
	Balance    *big.Int       `json:"balance"`
	AbciPubkey types.PubKey   `json:"abciPubkey"`
}

func NewValSortlist() *ValSortlist {
	return &ValSortlist{
		ValList: list.New(),
		Len:     0,
	}
}

func (valSortlist *ValSortlist) UpsertVal(valListItem *ValListItem, existFlag bool) {
	if existFlag {
		currentEle := valSortlist.ValList.Front()
		currentItem := valSortlist.ValList.Front().Value.(*ValListItem)
		for i := 0; i < valSortlist.Len; i++ {
			if bytes.Equal(currentItem.Signer.Bytes(), valListItem.Signer.Bytes()) {
				valSortlist.ValList.Remove(currentEle)
			}
		}
	}
	compareEle := valSortlist.ValList.Front()
	compareItem := compareEle.Value.(*ValListItem)
	for i := 0; i < valSortlist.Len; i++ {
		if valListItem.Balance.Cmp(compareItem.Balance) >= 0 {
			valSortlist.ValList.InsertBefore(valListItem, compareEle)
		}
	}
}

func (vallist *ValSortlist) RemoveVal(valListItem *ValListItem) {
	currentEle := vallist.ValList.Front()
	currentItem := vallist.ValList.Front().Value.(*ValListItem)
	for i := 0; i < vallist.Len; i++ {
		if bytes.Equal(currentItem.Signer.Bytes(), valListItem.Signer.Bytes()) {
			vallist.ValList.Remove(currentEle)
		}
	}
}

func (vallist *ValSortlist) GetTop100ValPubkey() map[types.PubKey]bool {
	pubKeyMap := make(map[types.PubKey]bool)
	currentEle := vallist.ValList.Front()
	currentItem := vallist.ValList.Front().Value.(*ValListItem)
	for i := 0; i < vallist.Len; i++ {
		if i == 100 {
			break
		}
		pubKeyMap[currentItem.AbciPubkey] = true
		currentEle = currentEle.Next()
		currentItem = currentEle.Value.(*ValListItem)
	}
	return pubKeyMap
}
