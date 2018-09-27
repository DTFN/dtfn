package app

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	emtTypes "github.com/tendermint/ethermint/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmTypes "github.com/tendermint/tendermint/types"
	"strings"
	"testing"
)

func TestUpsertValidatorTx(t *testing.T) {
//init the TMpubkeys, signer & beneficiary, strategy, ethmintapplication
	//TMpubkeys
	//pubkey:=ed25519.GenPrivKey().PubKey()
	pubkeylist:=[10]crypto.PubKey{}
	for i:=0;i<10;i++{
		pubkeylist[i]=ed25519.GenPrivKey().PubKey()
	}

	//signer & beneficiary
	var SignerList [10]common.Address
	var BeneList [10]common.Address
	SignerList[0]=common.HexToAddress("0x0000000000000000000000000000000000000001")
	SignerList[1]=common.HexToAddress("0x0000000000000000000000000000000000000002")
	SignerList[2]=common.HexToAddress("0x0000000000000000000000000000000000000003")
	SignerList[3]=common.HexToAddress("0x0000000000000000000000000000000000000004")
	SignerList[4]=common.HexToAddress("0x0000000000000000000000000000000000000005")
	SignerList[5]=common.HexToAddress("0x0000000000000000000000000000000000000006")
	SignerList[6]=common.HexToAddress("0x0000000000000000000000000000000000000007")
	SignerList[7]=common.HexToAddress("0x0000000000000000000000000000000000000008")
	SignerList[8]=common.HexToAddress("0x0000000000000000000000000000000000000009")
	SignerList[9]=common.HexToAddress("0x0000000000000000000000000000000000000010")
	BeneList[0]=common.HexToAddress("0x0000000000000000000000000000000000000011")
	BeneList[1]=common.HexToAddress("0x0000000000000000000000000000000000000012")
	BeneList[2]=common.HexToAddress("0x0000000000000000000000000000000000000013")
	BeneList[3]=common.HexToAddress("0x0000000000000000000000000000000000000014")
	BeneList[4]=common.HexToAddress("0x0000000000000000000000000000000000000015")
	BeneList[5]=common.HexToAddress("0x0000000000000000000000000000000000000016")
	BeneList[6]=common.HexToAddress("0x0000000000000000000000000000000000000017")
	BeneList[7]=common.HexToAddress("0x0000000000000000000000000000000000000018")
	BeneList[8]=common.HexToAddress("0x0000000000000000000000000000000000000019")
	BeneList[9]=common.HexToAddress("0x0000000000000000000000000000000000000020")

	//strategy
	//  ethmintapplication.Strategy.AccountMapList.MapList
	MapList:=make(map[string]*tmTypes.AccountMap)
	var PubAddress string
	for i:=0;i<10;i++ {
		PubAddress=strings.ToLower(hex.EncodeToString(pubkeylist[i].Address()))
		MapList[string(PubAddress)] = &tmTypes.AccountMap{
			//SignerList[i],
			//BeneList[i],
		}
	}
	//  ethmintapplication.Strategy.AccountMapList
	AML:=&tmTypes.AccountMapList{}
	AML.MapList=MapList

	//  ethmintapplication.Strategy.Validatorset.NextCandidateValidators
	//  ethmintapplication.Strategy.Validatorset
	//  ethmintapplication.Strategy.PosTable


	//  ethmintapplication.Strategy
	strategy:=&emtTypes.Strategy{}
	strategy.AccountMapList=AML
	strategy.PosTable=nil

	//ethmintapplication
	ethapp:=&EthermintApplication{}
	ethapp.strategy=strategy

	//test
	upsertFlag, err:=ethapp.UpsertValidatorTx(SignerList[0],300, BeneList[0], pubkeylist[0])
	require.NoError(t,err)
	require.Equal(t,false,upsertFlag)

	//upsertFlag, err=ethapp.UpsertValidatorTx(SignerList[0],300,BeneList[0],pubkeylist[0])


}

func TestRemoveValidatorTx(t *testing.T) {}

func TestComplicated(t *testing.T){}