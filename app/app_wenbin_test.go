package app

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	emtTypes "github.com/tendermint/ethermint/types"
	"strconv"
	"testing"
)

var pubkeylist [10]crypto.PubKey
var SignerList, BeneList [10]common.Address

func NewMockEthApplication(strategy *emtTypes.Strategy) (*EthermintApplication, error) {

	app := &EthermintApplication{
		strategy: strategy,
	}
	return app, nil
}

func initPubKey() {
	pubkeylist = [10]crypto.PubKey{}
	for i := 0; i < 10; i++ {
		pubkeylist[i] = ed25519.GenPrivKey().PubKey()
	}
	//signer & beneficiary
	for i := 0; i < 10; i++ {
		SignerList[i] = common.HexToAddress("0x00000000000000000000000000000000000000" + strconv.Itoa(i+10))
		BeneList[i] = common.HexToAddress("0x00000000000000000000000000000000000000" + strconv.Itoa(i+20))
	}
}

func TestUpsertValidator(t *testing.T) {
	initPubKey()

	strategy := emtTypes.NewStrategy()
	ethapp, err := NewMockEthApplication(strategy)

	if err != nil {
	} else {
		fmt.Println(ethapp.strategy.Receiver())
	}

}
