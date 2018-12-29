package app

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"github.com/ethereum/go-ethereum/core/state"
	"strings"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/green-element-chain/gelchain/types"
	"github.com/ethereum/go-ethereum/log"
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
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return big.NewInt(0)
	}
	balance := stateDB.GetBalance(addr)
	if balance.Cmp(amount) < 0 {
		amount = balance
	}
	balance = big.NewInt(0).Sub(balance, amount)
	stateDB.SetBalance(addr, balance)
	stateDB.Commit(false)
	return amount
}

type IApp interface {
	RemoveValidatorTx(signer common.Address) (bool, error)
	GetAccountMap(tmAddress string) *types.AccountMapItem
}

func (p *Punishment) DoPunish(app IApp, stateDB *state.StateDB, evidences []abciTypes.Evidence, accountMap *types.AccountMap) {
	for _, e := range evidences {
		var signer common.Address
		found := false

		for tmAddress, item := range accountMap.MapList {
			/*				pubKey:=item.PubKey
							tmPubKey,_:=tmTypes.PB2TM.PubKey(pubKey)*/
			if strings.EqualFold(string(e.Validator.Address), tmAddress) {
				signer = item.Signer
				found = true
				break
			}
		}
		if found {
			/*			tmAddress := strings.ToUpper(hex.EncodeToString(addr))
						accountMapItem := app.GetAccountMap(tmAddress)*/
			p.Punish(stateDB, signer)
			//To do
			//app.RemoveValidatorTx(signer)
		} else {
			log.Error(fmt.Sprintf("Fail to punish address %X. Evidence %v is too long ago?", e.Validator.Address, e))
		}
	}
}
