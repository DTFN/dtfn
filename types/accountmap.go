package types

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	cmn "github.com/tendermint/tendermint/libs/common"
	"io/ioutil"
	"math/big"
	"sync"
)

// AccountMap is an initial accountMap between tendermitn address and go-ethereum-address.
type AccountMap struct {
	Signer           common.Address `json:"signer"`
	SignerBalance    *big.Int       `json:"signerBalance"`
	BeneficiaryBonus *big.Int       `json:"beneficiaryBonus"`
	Beneficiary      common.Address `json:"beneficiary"`
	BlsKeyString     string         `json:"blsKeyString"`
}

// AccountMapList defines the initial list of AccountMap.
type AccountMapList struct {
	MapList map[string]*AccountMap `json:"accountmaplist"`

	FilePath string `json:"filePath"`
	mtx      sync.Mutex
}

// GenFilePV generates a new validator with randomly generated private key
// and sets the filePath, but does not call Save().
func (am *AccountMapList) GenAccountMapList(filePath string) *AccountMapList {
	am.FilePath = filePath
	am.Save()
	return am
}

// Save persists the FilePV to disk.
func (am *AccountMapList) Save() {
	am.mtx.Lock()
	defer am.mtx.Unlock()
	am.save()
}

func (am *AccountMapList) save() {
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
func AccountMapFromJSON(jsonBlob []byte) (*AccountMapList, error) {
	amlist := AccountMapList{}
	err := json.Unmarshal(jsonBlob, &amlist)
	if err != nil {
		return nil, err
	}

	return &amlist, err
}

// AccountMapFromJSON reads JSON data from a file and unmarshalls it into a AccountMapList.
func AccountMapFromFile(AccountMapFile string) (*AccountMapList, error) {
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
