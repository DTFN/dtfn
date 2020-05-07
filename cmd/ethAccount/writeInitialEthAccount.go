package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"io/ioutil"
	"math/big"
	"os"
	"sort"
	"strings"
)

var ethGenesisPath string
var ethSignerAccount [101]common.Address
var ethSignerAccountIndex int

type EthAccounts struct {
	EthAccounts     []string   `json:"ethAccounts"`
	EthBalances     []*big.Int `json:"ethBalances"`
	EthBeneficiarys []string   `json:"ethBeneficiarys"`
}

func main() {
	initParam()
	genesis, err := readEthGenesis()
	if err != nil {
		fmt.Println(err)
	}

	ethAccounts := &EthAccounts{}

	var addresses []string
	for address := range genesis.Alloc {
		addresses = append(addresses, address.String())
	}
	sort.Strings(addresses)

	for _, keyString := range addresses {
		ethAccounts.EthAccounts = append(
			ethAccounts.EthAccounts, keyString)
		ethAccounts.EthBalances = append(
			ethAccounts.EthBalances, genesis.Alloc[common.HexToAddress(keyString)].Balance)
		keyCopy := strings.ToLower(keyString)
		beneficaryStr := keyCopy[0:len(keyString)-2] + "01"
		ethAccounts.EthBeneficiarys = append(
			ethAccounts.EthBeneficiarys, beneficaryStr)
	}

	jsonEthAccounts, err := json.Marshal(ethAccounts)
	if err != nil {
		panic("get initial account failed")
	}

	err = ioutil.WriteFile("initial_eth_account.json", jsonEthAccounts, os.ModeAppend)
	if err != nil {
		return
	}
}

func initParam() {
	ethGenesisPath = os.Args[1]
}

func readEthGenesis() (*core.Genesis, error) {
	file, err := os.Open(ethGenesisPath)
	if err != nil {
		utils.Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	} else {
		loopIndex := 0
		ethSignerAccountIndex = 0
		for key, _ := range genesis.Alloc {
			ethSignerAccount[loopIndex] = key
			loopIndex++
			ethSignerAccountIndex++
		}
	}

	return genesis, err
}
