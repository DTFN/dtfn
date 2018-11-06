package types

import (
	"bytes"
	"container/list"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type ValSortlist struct {
	ValList *list.List
	Len     int
}

type ValListItem struct {
	Signer    common.Address `json:"signer"`
	Balance   *big.Int       `json:"balance"`
	TmAddress string         `json:"tmAddress"`
}

func NewValSortlist() *ValSortlist {
	return &ValSortlist{
		ValList: list.New(),
		Len:     0,
	}
}

func (valSortlist *ValSortlist) UpsertVal(valListItem *ValListItem, existFlag bool) {
	if valSortlist.Len == 0 {
		return
	}
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
	valSortlist.Len = valSortlist.Len + 1
}

func (vallist *ValSortlist) RemoveVal(valListItem *ValListItem) {
	currentEle := vallist.ValList.Front()
	currentItem := vallist.ValList.Front().Value.(*ValListItem)
	for i := 0; i < vallist.Len; i++ {
		if bytes.Equal(currentItem.Signer.Bytes(), valListItem.Signer.Bytes()) {
			vallist.ValList.Remove(currentEle)
		}
	}
	vallist.Len = vallist.Len - 1
}

func (vallist *ValSortlist) GetTopValTmAddress() map[string]bool {
	tmAddressMap := make(map[string]bool)
	currentEle := vallist.ValList.Front()
	currentItem := vallist.ValList.Front().Value.(*ValListItem)
	for i := 0; i < vallist.Len; i++ {
		if i == 100 {
			break
		}
		tmAddressMap[currentItem.TmAddress] = true
		currentEle = currentEle.Next()
		currentItem = currentEle.Value.(*ValListItem)
	}
	return tmAddressMap
}
