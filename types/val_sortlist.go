package types

import (
	"container/list"
)

type ValSortlist struct {
	ValList *list.List
	Len     int
}

func NewValSortlist() *ValSortlist {
	return &ValSortlist{
		ValList: list.New(),
		Len:     0,
	}
}

func (valSortlist *ValSortlist) UpsertVal(posItem *PosItem, existFlag bool) {
	if existFlag {
		valSortlist.ValList.PushFront(posItem)
	} else {
		valSortlist.ValList.PushFront(posItem)
		valSortlist.Len = valSortlist.Len + 1
	}
}

func (vallist *ValSortlist) RemoveVal(posItem *PosItem) {

}
