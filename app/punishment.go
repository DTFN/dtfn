package app

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"github.com/ethereum/go-ethereum/core/state"
	"strings"
	"encoding/hex"
	tmTypes "github.com/tendermint/tendermint/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/green-element-chain/gelchain/types"
)

type Punishment struct {
	amountStrategy     AmountStrategy
	subBalanceStrategy SubBalanceStrategy
	stateDB            *state.StateDB
}

func NewPunishment(as AmountStrategy, ss SubBalanceStrategy, stateDB *state.StateDB) *Punishment {
	punishment := &Punishment{amountStrategy: as, subBalanceStrategy: ss, stateDB: stateDB}
	return punishment
}

func (p *Punishment) Punish(byzantine common.Address) error {
	as := p.amountStrategy
	ss := p.subBalanceStrategy
	ss.subBalance(p.stateDB, byzantine, as.amount(p.stateDB, byzantine))
	return nil
}

type AmountStrategy interface {
	amount(stateDB *state.StateDB, byzantine common.Address) *big.Int
}

type FixedAmountStrategy struct {
	fixedAmount *big.Int
}

func (f *FixedAmountStrategy) amount(stateDB *state.StateDB, byzantine common.Address) *big.Int {
	return f.fixedAmount
}

type PercentAmountStrategy struct {
	percent uint
}

func (f *PercentAmountStrategy) amount(stateDB *state.StateDB, byzantine common.Address) *big.Int {
	if f.percent > 100 {
		f.percent = 100
	}
	if f.percent < 0 {
		f.percent = 0
	}
	var p float64 = float64(f.percent) / 100.0
	balance := stateDB.GetBalance(byzantine)
	amount := big.NewFloat(0).Mul(big.NewFloat(p), big.NewFloat(float64(balance.Int64())))
	i, _ := amount.Int64()
	return big.NewInt(i)
}

type SubBalanceStrategy interface {
	subBalance(stateDB *state.StateDB, byzantine common.Address, balance *big.Int)
}

type BurnStrategy struct {
}

func (s *BurnStrategy) subBalance(stateDB *state.StateDB, byzantine common.Address, balance *big.Int) {
	subBalance(stateDB, byzantine, balance)
}

type TransferStrategy struct {
	transferTo common.Address
}

func (s *TransferStrategy) subBalance(stateDB *state.StateDB, addr common.Address, amount *big.Int) {
	amount = subBalance(stateDB, addr, amount)
	if amount.Cmp(big.NewInt(0)) > 0 {
		stateDB.AddBalance(s.transferTo, amount)
	}
}

func subBalance(stateDB *state.StateDB, addr common.Address, amount *big.Int) *big.Int {
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return big.NewInt(0)
	}
	balance := stateDB.GetBalance(addr)
	if balance.Cmp(amount) < 0 {
		amount = balance
	}
	balance = big.NewInt(0).Sub(balance, amount)
	stateDB.SetBalance(addr, balance)
	return amount
}

type IApp interface {
	RemoveValidatorTx(signer common.Address) (bool, error)
	GetAccountMap(tmAddress string) *types.AccountMap
}

func (p *Punishment) DoPunish(app IApp, evidences []abciTypes.Evidence) {
	for _, e := range evidences {
		pubkey, _ := tmTypes.PB2TM.PubKey(e.Validator.PubKey)
		tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
		accountMap := app.GetAccountMap(tmAddress)
		signer := accountMap.Signer
		p.Punish(signer)
		app.RemoveValidatorTx(signer)
	}
}
