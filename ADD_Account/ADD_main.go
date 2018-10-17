package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	add_utils "github.com/regcostajr/go-web3/Add_More_Accounts/ADD_Util"
	"os"
)

// 以下分别为：Genesis.json原路径, 更改后需要输出的路径
var ethGenesisnewpath string
var address []common.Address

func main() {
	//1,初始化os.Arg输入参数
	initParam()

	//2,扫描keystore文件路径下所有accounts文件，取出地址列表address
	address, err := add_utils.ScanAccounts()

	//3,100个账户信息放入core.Genesis.alloc中，注意balance总额度不变
	genesis_new := add_utils.ModifyEthGenesis(address)

	//4,将genesis_new 转成.json文件放回原路径
	err = add_utils.HandleJson(genesis_new, ethGenesisnewpath)
	if err != nil {
		fmt.Print(err)
	}
}

func initParam() {
	ethGenesisnewpath = os.Args[1]
}
