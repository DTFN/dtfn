package app

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	emtTypes "github.com/tendermint/ethermint/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmTypes "github.com/tendermint/tendermint/types"
	"strconv"
	"testing"
)

var pubkeylist [10]crypto.PubKey
var SignerList, BeneList [10]common.Address

func TestUpsertValidator(t *testing.T) {
	initPubKey()

	strategy := emtTypes.NewStrategy()
	ethapp, err := NewMockEthApplication(strategy)
	require.NoError(t, err)

	MapList := make(map[string]*tmTypes.AccountMap)
	AML := &tmTypes.AccountMapList{MapList: MapList}

	ethapp.strategy.AccountMapList = AML

	upsertFlag, err := ethapp.UpsertValidatorTx(SignerList[0], 300, BeneList[0], pubkeylist[0])
	require.NoError(t, err)
	require.Equal(t, false, upsertFlag)
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

func NewMockEthApplication(strategy *emtTypes.Strategy) (*EthermintApplication, error) {
	app := &EthermintApplication{
		strategy: strategy,
	}
	return app, nil
}
