package app

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/green-element-chain/gelchain/app/mock_log"
	emtTypes "github.com/green-element-chain/gelchain/types"
	"github.com/stretchr/testify/require"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmLog "github.com/tendermint/tendermint/libs/log"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

var pubkeylist [10]crypto.PubKey
var SignerList, BeneList [10]common.Address

//about app.go
func TestGetStrategy(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, err := NewMockEthApplication(strategy,mock_logger)
	require.NoError(t, err)
	require.Equal(t,big.NewInt(20000),ethapp.strategy.TotalBalance)
}


//about utils.go
func TestReceiver(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	Address:=ethapp.Receiver()
	require.Equal(t,common.HexToAddress("0000000000000000000000000000000000000002"),Address)
}

func TestSetThreShold(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	ethapp.SetThreShold(big.NewInt(1000))
	require.Equal(t,big.NewInt(1000),ethapp.strategy.CurrRoundValData.PosTable.Threshold)
}

func TestUpsertPosItem(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	PubKey := abciTypes.PubKey{
		Type:	 "ed25519",
		Data: []byte("lSk6hpSsP+Vpi/yfNFbfqK4x99jx1zTk"),
	}

	pubKey,_:= tmTypes.PB2TM.PubKey(PubKey)
	pubkstr:=strings.ToLower(pubKey.Address().String())
	fmt.Print(pubkstr)

	upsertFlag, _:=ethapp.UpsertPosItem(common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b"),
		big.NewInt(10000), common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b"),PubKey)
	require.Equal(t,true,upsertFlag)
	require.NoError(t,nil)
}

func TestRemovePosItem(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	removeflag, _:=ethapp.RemovePosItem(common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b"))
	require.Error(t,fmt.Errorf("address not existed in the postable"))
	require.Equal(t,false, removeflag)
}

func TestUpsertValidator(t *testing.T) {
	//Controller
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	//info 的结果 是 Call
	mock_logger.EXPECT().Info("You are upsert ValidatorTxing").Return()
	mock_logger.EXPECT().Info("nil validator pubkey or bls pubkey").Return()
	initPubKey()

	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, err1 := NewMockEthApplication(strategy,mock_logger)
	require.NoError(t, err1)

	MapList := make(map[string]*emtTypes.AccountMap)
	AML := &emtTypes.AccountMapList{MapList: MapList}

	ethapp.strategy.CurrRoundValData.AccountMapList = AML
	upsertFlag, err2 := ethapp.UpsertValidatorTx(SignerList[0], big.NewInt(1), big.NewInt(300), BeneList[0], pubkeylist[0],"")
	require.Error(t,errors.New("nil validator pubkey or bls pubkey"),err2)
	require.Equal(t, false,upsertFlag)
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

func NewMockEthApplication(strategy *emtTypes.Strategy,logger tmLog.Logger) (*EthermintApplication, error) {
	//mockLogger
	app := &EthermintApplication{
		strategy: strategy,
		logger: logger,
	}
	return app, nil
}

func TestRemoveValidatorTx(t *testing.T) {
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	mock_logger.EXPECT().Info("You are upsert ValidatorTxing").Return()
	mock_logger.EXPECT().Info("add Validator Tx success").Return()

	mock_logger.EXPECT().Info("You are removeValidatorTx").Return()
	mock_logger.EXPECT().Info("can not remove validator for error-tolerant").Return()

	initPubKey()

	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, err1 := NewMockEthApplication(strategy,mock_logger)
	require.NoError(t, err1)

	MapList := make(map[string]*emtTypes.AccountMap)
	AML := &emtTypes.AccountMapList{MapList: MapList}

	ethapp.strategy.CurrRoundValData.AccountMapList = AML

	upsertFlag, _ := ethapp.RemoveValidatorTx(SignerList[0])
	require.Error(t, errors.New("can not remove validator for error-tolerant"))
	require.Equal(t, 0, len(ethapp.strategy.CurrRoundValData.AccountMapList.MapList))
	require.Equal(t, false, upsertFlag)
}

func TestenterInitial(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	ResBlock:=ethapp.enterInitial(1)
	require.Equal(t,abciTypes.ResponseEndBlock{},ResBlock)

	var Validators=[]*abciTypes.Validator{
		{	Address: []byte("43A280B075C15EEA8EDE123ED84462C260F780CC"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("lSk6hpSsP+Vpi/yfNFbfqK4x99jx1zTkf7On60ES3I4="),
			},
			Power: 1,},
		{
			Address: []byte("E431AE48F0F9894E7FBE06CF5CCF66B326D7439F"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("DAgy3l3jPF8L24KBTs7oJfyduihcBoiOOYIstEMx9VY="),
			},
			Power: 2,},
	}
	ethapp.strategy.CurrRoundValData.InitialValidators=Validators
	ethapp.strategy.SetValidators(Validators)
	ResBlock=ethapp.enterInitial(1)
	require.Equal(t,2,ethapp.strategy.CurrRoundValData.CurrentValidators)
}

func TestenterSelectValidators(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	var seed []byte
	var height int64
	ResponseEndBlock:=ethapp.enterSelectValidators(seed, height)
	require.Equal(t,abciTypes.ResponseEndBlock{},ResponseEndBlock)

	var Validators=[]*abciTypes.Validator{
		{	Address: []byte("43A280B075C15EEA8EDE123ED84462C260F780CC"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("lSk6hpSsP+Vpi/yfNFbfqK4x99jx1zTkf7On60ES3I4="),
			},
			Power: 1,},
		{
			Address: []byte("E431AE48F0F9894E7FBE06CF5CCF66B326D7439F"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("DAgy3l3jPF8L24KBTs7oJfyduihcBoiOOYIstEMx9VY="),
			},
			Power: 2,},
		{	Address: []byte("84A280B075C15EEA8EDE123ED84462C260F780CC"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("9996hpSsP+Vpi/yfNFbfqK4x99jx1zTkf7On60ES3I4="),
			},
			Power: 3,},
		{	Address: []byte("0000000000000000000000000000000000000001"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("6666hpSsP+Vpi/yfNFbfqK4x99jx1zTkf7On60ES3I4="),
			},
			Power: 4,},
	}
	ethapp.strategy.CurrRoundValData.CurrCandidateValidators=Validators
	ResponseEndBlock=ethapp.enterSelectValidators(seed, 10)
	require.Equal(t,4,len(ResponseEndBlock.ValidatorUpdates))
}

func TestblsValidators(t *testing.T){
	ctl := gomock.NewController(t)
	mock_logger := mock_log.NewMockLogger(ctl)
	initPubKey()
	strategy := emtTypes.NewStrategy(big.NewInt(20000))
	ethapp, _ := NewMockEthApplication(strategy,mock_logger)
	var Validators=[]*abciTypes.Validator{
		{	Address: []byte("43A280B075C15EEA8EDE123ED84462C260F780CC"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("lSk6hpSsP+Vpi/yfNFbfqK4x99jx1zTkf7On60ES3I4="),
			},
			Power: 1,},
		{
			Address: []byte("E431AE48F0F9894E7FBE06CF5CCF66B326D7439F"),
			PubKey: abciTypes.PubKey{
				Type:	 "tendermint/PubKeyEd25519",
				Data: []byte("DAgy3l3jPF8L24KBTs7oJfyduihcBoiOOYIstEMx9VY="),
			},
			Power: 2,},
	}
	var acm1=emtTypes.AccountMap{
		common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b"),
		big.NewInt(100000),
		big.NewInt(1000),
		common.HexToAddress("0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b"),
		"0",
	}

	var acm2=emtTypes.AccountMap{
		common.HexToAddress("0x002f4e1ed26d8e8491046ac2c2faff8df1be470e"),
		big.NewInt(100000),
		big.NewInt(1000),
		common.HexToAddress("0x002f4e1ed26d8e8491046ac2c2faff8df1be470e"),
		"1",
	}
	var Maplist =map[string]*emtTypes.AccountMap{
		"0xd84c6fb02305c9ea2f20f97e0cccea4e54f9014b":&acm1,
		"0x002f4e1ed26d8e8491046ac2c2faff8df1be470e":&acm2,
	}
	var AccountMapList=emtTypes.AccountMapList{
		MapList:Maplist,
	}
	ethapp.strategy.CurrRoundValData.AccountMapList=&AccountMapList
	ethapp.strategy.CurrRoundValData.InitialValidators=Validators
	ethapp.strategy.SetValidators(Validators)
	ResponseEndBlock:=ethapp.blsValidators(1)
	require.Equal(t,4,len(ethapp.strategy.CurrRoundValData.CurrentValidators))
	require.Equal(t,ethapp.strategy.CurrRoundValData.CurrentValidators[0].Address,ResponseEndBlock.ValidatorUpdates[0].Address)
}