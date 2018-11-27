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
	"fmt"
)

type Punishment struct {
	amountStrategy     AmountStrategy
	subBalanceStrategy SubBalanceStrategy
}

func NewPunishment(as AmountStrategy, ss SubBalanceStrategy) *Punishment {
	punishment := &Punishment{amountStrategy: as, subBalanceStrategy: ss}
	return punishment
}

func (p *Punishment) Punish(stateDB *state.StateDB, byzantine common.Address) error {
	as := p.amountStrategy
	ss := p.subBalanceStrategy
	ss.subBalance(stateDB, byzantine, as.amount(stateDB, byzantine))
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

type Percent100AmountStrategy struct {
}

func (f *Percent100AmountStrategy) amount(stateDB *state.StateDB, byzantine common.Address) *big.Int {
	return stateDB.GetBalance(byzantine)
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
	fmt.Println("addr:", addr.Hex())
	fmt.Println("1", amount)
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return big.NewInt(0)
	}
	balance := stateDB.GetBalance(addr)
	fmt.Println("2", balance)
	if balance.Cmp(amount) < 0 {
		amount = balance
	}
	balance = big.NewInt(0).Sub(balance, amount)
	fmt.Println("3", balance)
	fmt.Println("4", amount)
	stateDB.SetBalance(addr, balance)
	stateDB.Commit(false)
	return amount
}

type IApp interface {
	RemoveValidatorTx(signer common.Address) (bool, error)
	GetAccountMap(tmAddress string) *types.AccountMap
}

func (p *Punishment) DoPunish(app IApp, stateDB *state.StateDB, evidences []abciTypes.Evidence, vs []*abciTypes.Validator) {
	for _, e := range evidences {
		pk := e.Validator.PubKey

		for _, v := range vs {
			fmt.Println("vvv:", string(e.Validator.Address), string(v.Address))
			if strings.EqualFold(string(e.Validator.Address), string(v.Address)) {
				pk = v.PubKey
				break
			}
		}
		pubkey, e := tmTypes.PB2TM.PubKey(pk)
		if e != nil {
			continue
		}
		fmt.Println("punish:", pubkey)
		if pubkey != nil {
			tmAddress := strings.ToLower(hex.EncodeToString(pubkey.Address()))
			accountMap := app.GetAccountMap(tmAddress)
			signer := accountMap.Signer
			p.Punish(stateDB, signer)
			b, _ := app.RemoveValidatorTx(signer)
			fmt.Println("b", b)
		}
	}
}