package types

import (
	"bytes"
	"container/list"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type ValSortlist struct {
	ValList *list.List `json:"valList"`
	Len     int        `json:"len"`
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
		valSortlist.ValList.PushFront(valListItem)
		valSortlist.Len = 1
		return
	}
	if existFlag {
		currentEle := valSortlist.ValList.Front()
		currentItem := valSortlist.ValList.Front().Value.(*ValListItem)
		for i := 0; i < valSortlist.Len-1; i++ {
			if bytes.Equal(currentItem.Signer.Bytes(), valListItem.Signer.Bytes()) {
				valSortlist.ValList.Remove(currentEle)
				valSortlist.Len = valSortlist.Len - 1
				break
			}
			currentEle = currentEle.Next()
			currentItem = currentEle.Value.(*ValListItem)
		}
	}
	compareEle := valSortlist.ValList.Front()
	compareItem := compareEle.Value.(*ValListItem)
	for i := 0; i < valSortlist.Len; i++ {
		if valListItem.Balance.Cmp(compareItem.Balance) >= 0 {
			valSortlist.ValList.InsertBefore(valListItem, compareEle)
			break
		} else if i == valSortlist.Len-1 {
			valSortlist.ValList.InsertAfter(valListItem,compareEle)
		} else {
			compareEle = compareEle.Next()
			compareItem = compareEle.Value.(*ValListItem)
		}
	}
	valSortlist.Len = valSortlist.Len + 1
}

func (vallist *ValSortlist) RemoveVal(valListItem *ValListItem) {
	currentEle := vallist.ValList.Front()
	currentItem := vallist.ValList.Front().Value.(*ValListItem)
	for i := 0; i < vallist.Len-1; i++ {
		if bytes.Equal(currentItem.Signer.Bytes(), valListItem.Signer.Bytes()) {
			vallist.ValList.Remove(currentEle)
			break
		}
		currentEle = currentEle.Next()
		currentItem = currentEle.Value.(*ValListItem)
	}
	vallist.Len = vallist.Len - 1
}

func (vallist *ValSortlist) GetTopValTmAddress() map[string]bool {
	tmAddressMap := make(map[string]bool)
	currentEle := vallist.ValList.Front()
	currentItem := vallist.ValList.Front().Value.(*ValListItem)
	tmAddressMap[currentItem.TmAddress] = true
	for i := 0; i < vallist.Len; i++ {
		if i == 100 {
			break
		}
		tmAddressMap[currentItem.TmAddress] = true
		if i == vallist.Len-1 {
			break
		}
		currentEle = currentEle.Next()
		currentItem = currentEle.Value.(*ValListItem)
	}
	return tmAddressMap
}
