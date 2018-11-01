package app

import (
	"bytes"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	emtTypes "github.com/green-element-chain/gelchain/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

var pubkeylist [10]crypto.PubKey
var SignerList, BeneList [10]common.Address

func TestUpsertValidator(t *testing.T) {
	initPubKey()

	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, err := NewMockEthApplication(strategy)
	require.NoError(t, err)

	MapList := make(map[string]*tmTypes.AccountMap)
	AML := &tmTypes.AccountMapList{MapList: MapList}

	ethapp.strategy.AccountMapList = AML

	upsertFlag, err := ethapp.UpsertValidatorTx(SignerList[0], big.NewInt(300), BeneList[0], pubkeylist[0],"")
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

func TestRemoveValidatorTx(t *testing.T) {
	initPubKey()

	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, err := NewMockEthApplication(strategy)
	require.NoError(t, err)

	MapList := make(map[string]*tmTypes.AccountMap)
	AML := &tmTypes.AccountMapList{MapList: MapList}

	ethapp.strategy.AccountMapList = AML

	upsertFlag, err := ethapp.UpsertValidatorTx(SignerList[0], big.NewInt(1000), BeneList[0], pubkeylist[0],"")
	require.NoError(t, err)
	require.Equal(t, false, upsertFlag)

	upsertFlag, err = ethapp.RemoveValidatorTx(SignerList[0])
	require.Equal(t, 0, len(ethapp.strategy.AccountMapList.MapList))
}

func TestComplicated(t *testing.T) {
	initPubKey()

	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, err := NewMockEthApplication(strategy)
	require.NoError(t, err)

	MapList := make(map[string]*tmTypes.AccountMap)
	AML := &tmTypes.AccountMapList{MapList: MapList}

	ethapp.strategy.AccountMapList = AML

	//Complicated_UpsertValidatorTX & Generate NextCandidateValidators
	upsertFlag, err := ethapp.UpsertValidatorTx(SignerList[0], big.NewInt(300), BeneList[0], pubkeylist[0],"")
	require.NoError(t, err)
	require.Equal(t, false, upsertFlag)

	upsertFlag, err = ethapp.RemoveValidatorTx(SignerList[0])
	require.Equal(t, 0, len(ethapp.strategy.AccountMapList.MapList))

	upsertFlag, err = ethapp.UpsertValidatorTx(SignerList[1], big.NewInt(300), BeneList[0], pubkeylist[1],"")
	require.Equal(t, SignerList[1], ethapp.strategy.AccountMapList.MapList[strings.ToLower(hex.EncodeToString(pubkeylist[1].Address()))].Signer)

	//Complicated_RemoveValidatorTx
	//找到tmAddress，另这个的signer与输入相等
	var tmAddress string
	for k, v := range ethapp.strategy.AccountMapList.MapList {
		if bytes.Equal(v.Signer.Bytes(), SignerList[1].Bytes()) {
			tmAddress = k
			break
		}
	}
	require.Equal(t, SignerList[1], ethapp.strategy.AccountMapList.MapList[tmAddress].Signer)
	require.Equal(t, []byte(pubkeylist[1].Address()), ethapp.strategy.ValidatorSet.NextCandidateValidators[0].Address)
	upsertFlag, err = ethapp.RemoveValidatorTx(SignerList[1])
	require.Equal(t, 0, len(ethapp.strategy.AccountMapList.MapList))
	require.Equal(t, 0, len(ethapp.strategy.ValidatorSet.NextCandidateValidators))
}
