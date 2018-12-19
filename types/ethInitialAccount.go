package types

import (
	"encoding/json"
	"fmt"
	cmn "github.com/tendermint/tendermint/libs/common"
	"io/ioutil"
	"math/big"
)

type EthAccounts struct {
	EthAccounts     []string   `json:"ethAccounts"`
	EthBalances     []*big.Int `json:"ethBalances"`
	EthBeneficiarys []string   `json:"ethBeneficiarys"`
}

// GetInitialEthAccountFromFile reads JSON data from a file and unmarshalls it into a initial eth accounts.
func GetInitialEthAccountFromFile(EthAccountsPath string) (*EthAccounts, error) {
	jsonBlob, err := ioutil.ReadFile(EthAccountsPath)
	if err != nil {
		return nil, cmn.ErrorWrap(err, "Couldn't read AccountMap File")
	}
	ethAccounts, err := EthAccountsFromJSON(jsonBlob)
	if err != nil {
		return nil, cmn.ErrorWrap(err, fmt.Sprintf("Error reading GenesisDoc at %v", EthAccountsPath))
	}
	return ethAccounts, nil
}

// EthAccountsFromJSON unmarshalls JSON data into a eth accounts.
func EthAccountsFromJSON(jsonBlob []byte) (*EthAccounts, error) {
	ethAccounts := EthAccounts{}
	err := json.Unmarshal(jsonBlob, &ethAccounts)
	if err != nil {
		return nil, err
	}

	return &ethAccounts, err
}
