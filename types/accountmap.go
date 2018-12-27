package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	cmn "github.com/tendermint/tendermint/libs/common"
	"io/ioutil"
	"sync"
)

// AccountMapItem connects tm address with eth address and blsPubKey
type AccountMapItem struct {
	Signer           common.Address `json:"signer"`
	Beneficiary      common.Address `json:"beneficiary"`
	BlsKeyString     string         `json:"blsKey_string"`
}

func (accountMapItem *AccountMapItem) Copy() *AccountMapItem {
	return &AccountMapItem{
		accountMapItem.Signer,
		accountMapItem.Beneficiary,
		accountMapItem.BlsKeyString,
	}
}

type AccountMap struct {
	MapList map[string]*AccountMapItem `json:"map_list"`

	FilePath string `json:"file_path"`
	mtx      sync.Mutex
}

func (am *AccountMap) Copy() *AccountMap {
	newMapList:= map[string]*AccountMapItem{}
	for k,v:=range am.MapList{
		newMapList[k]=v.Copy()
	}
	return &AccountMap{
		MapList:newMapList,
		FilePath:am.FilePath,
	}
}

// GenFilePV generates a new validator with randomly generated private key
// and sets the filePath, but does not call Save().
func (am *AccountMap) GenAccountMapList(filePath string) *AccountMap {
	am.FilePath = filePath
	am.Save()
	return am
}

// Save persists the FilePV to disk.
func (am *AccountMap) Save() {
	am.mtx.Lock()
	defer am.mtx.Unlock()
	am.save()
}

func (am *AccountMap) save() {
	outFile := am.FilePath
	if outFile == "" {
		panic("Cannot save AccountMap: filePath not set")
	}
	jsonBytes, err := json.Marshal(am)
	if err != nil {
		panic(err)
	}
	err = cmn.WriteFile(outFile, jsonBytes, 0600)
	if err != nil {
		panic(err)
	}
}

// AccountMapFromJSON unmarshalls JSON data into a AccountMapList.
func AccountMapFromJSON(jsonBlob []byte) (*AccountMap, error) {
	amlist := AccountMap{}
	err := json.Unmarshal(jsonBlob, &amlist)
	if err != nil {
		return nil, err
	}

	return &amlist, err
}

// AccountMapFromJSON reads JSON data from a file and unmarshalls it into a AccountMapList.
func AccountMapFromFile(AccountMapFile string) (*AccountMap, error) {
	jsonBlob, err := ioutil.ReadFile(AccountMapFile)
	if err != nil {
		return nil, cmn.ErrorWrap(err, "Couldn't read AccountMap File")
	}
	amlist, err := AccountMapFromJSON(jsonBlob)
	if err != nil {
		return nil, cmn.ErrorWrap(err, fmt.Sprintf("Error reading GenesisDoc at %v", AccountMapFile))
	}
	return amlist, nil
}
