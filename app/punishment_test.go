package app

import (
	"testing"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/stretchr/testify/assert"
	"github.com/golang/mock/gomock"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/green-element-chain/gelchain/ethereum"
	types2 "github.com/tendermint/tendermint/types"
	gelTypes "github.com/green-element-chain/gelchain/types"
	"strings"
	"encoding/hex"
)

var (
	stateDB    *state.StateDB
	byzantine  = common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0")
	transferTo = common.HexToAddress("0x7777777777777777777777777777777777777777")
)

func Before(initBalance int64) {
	diskDB, _ := ethdb.NewMemDatabase()
	stateDB, _ = state.New(common.Hash{}, state.NewDatabase(diskDB))
	stateDB.SetBalance(byzantine, big.NewInt(initBalance))
	stateDB.SetBalance(transferTo, big.NewInt(0))
}

func TestBurnFixed5k(t *testing.T) {
	Before(1000000000)
	amountStrategy := &FixedAmountStrategy{fixedAmount: big.NewInt(5000)}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy).Punish(byzantine)
	assert.Equal(t, big.NewInt(1000000000 - 5000).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestBurnFixed150m(t *testing.T) {
	Before(1000000000)
	amountStrategy := &FixedAmountStrategy{fixedAmount: big.NewInt(1500000000)}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy).Punish(byzantine)
	assert.Equal(t, big.NewInt(0).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestBurnFixedNeg5k(t *testing.T) {
	Before(1000000000)
	amountStrategy := &FixedAmountStrategy{fixedAmount: big.NewInt(-5000)}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy).Punish(byzantine)
	assert.Equal(t, big.NewInt(1000000000).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestBurnPercent50(t *testing.T) {
	Before(1000000000)
	amountStrategy := &PercentAmountStrategy{percent: 50}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy, stateDB).Punish(byzantine)
	assert.Equal(t, big.NewInt(1000000000 * 0.5).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestBurnPercent49(t *testing.T) {
	Before(7)
	amountStrategy := &PercentAmountStrategy{percent: 47}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy, stateDB).Punish(byzantine)
	assert.Equal(t, big.NewInt(4).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestBurnPercent100(t *testing.T) {
	Before(1000000000)
	amountStrategy := &PercentAmountStrategy{percent: 100}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy, stateDB).Punish(byzantine)
	assert.Equal(t, big.NewInt(0).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestBurnPercent200(t *testing.T) {
	Before(1000000000)
	amountStrategy := &PercentAmountStrategy{percent: 200}
	subBalanceStrategy := &BurnStrategy{}
	NewPunishment(amountStrategy, subBalanceStrategy, stateDB).Punish(byzantine)
	assert.Equal(t, big.NewInt(0).Int64(), stateDB.GetBalance(byzantine).Int64())
}

func TestTransferBurnPercent50(t *testing.T) {
	Before(1000000000)
	amountStrategy := &PercentAmountStrategy{percent: 50}
	subBalanceStrategy := &TransferStrategy{transferTo: transferTo}
	NewPunishment(amountStrategy, subBalanceStrategy, stateDB).Punish(byzantine)
	assert.Equal(t, big.NewInt(1000000000 * 0.5).Int64(), stateDB.GetBalance(byzantine).Int64())
	assert.Equal(t, big.NewInt(1000000000 * 0.5).Int64(), stateDB.GetBalance(transferTo).Int64())
}

var input = `{
  "pub_key":{
    "type":"tendermint/PubKeyEd25519",
    "value":"q/7QL3skC/rvTYRXOO9I5y+RWOhahr9WjyNHkcf8OQ8="
  },
  "beneficiary":"0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"
}`

func TestDoPunish(t *testing.T) {
	Before(10000)

	data, _ := ethereum.MarshalTxData(input)
	tmPubkey := data.Pv.PubKey
	pbPubkey := types2.TM2PB.PubKey(tmPubkey)

	ctrl := gomock.NewController(t)
	app := NewMockIApp(ctrl)

	app.EXPECT().GetAccountMap(strings.ToLower(
		hex.EncodeToString(tmPubkey.Address()))).Return(
		&gelTypes.AccountMap{Signer: common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0")})

	app.EXPECT().RemoveValidatorTx(byzantine).Return(true, nil)

	amountStrategy := &PercentAmountStrategy{percent: 100}
	subBalanceStrategy := &BurnStrategy{}
	evidences := make([]types.Evidence, 0)
	evidences = append(evidences, types.Evidence{Validator: types.Validator{PubKey: pbPubkey}})
	assert.Equal(t, big.NewInt(10000).Int64(), stateDB.GetBalance(byzantine).Int64())
	NewPunishment(amountStrategy, subBalanceStrategy, stateDB).DoPunish(app, evidences)
	assert.Equal(t, big.NewInt(0).Int64(), stateDB.GetBalance(byzantine).Int64())
}
