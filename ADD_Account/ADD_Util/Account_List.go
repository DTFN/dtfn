package ADD_Util

import (
	"fmt"
	"github.com/regcostajr/go-web3"
	"github.com/regcostajr/go-web3/providers"
)

func ListAddress() ([]string, error) {
	web3 := initClient()
	accounts, err := web3.Personal.ListAccounts()
	fmt.Printf("accounts:", accounts)
	if err != nil {
		fmt.Print(err)
	}
	return accounts, err
}

func initClient() *web3.Web3 {
	ethClient := web3.NewWeb3(providers.NewHTTPProvider("127.0.0.1:8545", 10, false))
	return ethClient
}
