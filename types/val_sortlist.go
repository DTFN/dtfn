package types

import "container/list"

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

func (vallist *ValSortlist) UpsertVal() {

}

func (vallist *ValSortlist) RemoveVal() {

}
