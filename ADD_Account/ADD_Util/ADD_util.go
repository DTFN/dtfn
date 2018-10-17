package ADD_Util

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"io/ioutil"
	"math/big"
)

func ScanAccounts() ([]common.Address, error) {
	//获取地址string表示 在keystore文件目录中
	account, err := ListAddress()
	if err != nil {
		fmt.Print(err)
	}
	var hexaddress []common.Address
	for i := 1; i < len(account); i++ {
		hexaddress = append(hexaddress, common.HexToAddress(account[i]))
	}
	return hexaddress, err
}

func ModifyEthGenesis(address []common.Address) *core.Genesis {
	genesis := new(core.Genesis)
	genesis.Alloc = make(map[common.Address]core.GenesisAccount)
	//genesis.Alloc插入新的值
	for i := 0; i < len(address); i++ {
		GA := core.GenesisAccount{}
		A := big.NewInt(1000000000000000)
		B := big.NewInt(1000000000000000000)
		B_sum := big.NewInt(1)
		B_sum = B_sum.Mul(A, B)
		GA.Balance = B_sum //10000000000000000000000000000000000*10/100 根据总额改动
		genesis.Alloc[address[i]] = GA
	}
	return genesis
}

func HandleJson(genesis_new *core.Genesis, ethGenesisnewpath string) error {
	jsresult, err := genesis_new.MarshalJSON()
	//jsresult, err := json.Marshal(genesis_new)
	if err != nil {
		return err
	}
	//回写新genesis.json
	err = ioutil.WriteFile(ethGenesisnewpath, jsresult, 0644)
	return err
}
