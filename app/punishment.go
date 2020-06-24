package app

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"github.com/ethereum/go-ethereum/core/state"
	"strings"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"github.com/DTFN/gelchain/types"
	"github.com/ethereum/go-ethereum/log"
	"fmt"
	"encoding/hex"
)

type Punishment struct {
	AmountStrategy     AmountStrategy
	SubBalanceStrategy SubBalanceStrategy
}

func NewPunishment(as AmountStrategy, ss SubBalanceStrategy) *Punishment {
	punishment := &Punishment{AmountStrategy: as, SubBalanceStrategy: ss}
	return punishment
}

func (p *Punishment) Punish(stateDB *state.StateDB, byzantine common.Address) error {
	as := p.AmountStrategy
	ss := p.SubBalanceStrategy
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

func (f Percent100AmountStrategy) amount(stateDB *state.StateDB, byzantine common.Address) *big.Int {
	return stateDB.GetBalance(byzantine)
}

type SubBalanceStrategy interface {
	subBalance(stateDB *state.StateDB, byzantine common.Address, balance *big.Int)
}

type BurnStrategy struct {
}

func (s BurnStrategy) subBalance(stateDB *state.StateDB, byzantine common.Address, balance *big.Int) {
	subBalance(stateDB, byzantine, balance)
}

type TransferStrategy struct {
	transferTo common.Address
}

func (s TransferStrategy) subBalance(stateDB *state.StateDB, addr common.Address, amount *big.Int) {
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

func (p *Punishment) DoPunish(stateDB *state.StateDB, strategy *types.Strategy, evidences []abciTypes.Evidence, coinbase common.Address, currentHeight int64) {
	if transferStrategy, ok := p.SubBalanceStrategy.(TransferStrategy); ok {
		transferStrategy.transferTo = coinbase
	}
	for _, e := range evidences {
		signer, found := strategy.NextEpochValData.PosTable.TmAddressToSignerMap[strings.ToUpper(hex.EncodeToString(e.Validator.Address))]
		if found {
			p.Punish(stateDB, signer)
			log.Info(fmt.Sprintf("evil signer %v got slashed because of Evidence %v", signer, e))
			_, found := strategy.NextEpochValData.PosTable.PosItemMap[signer]
			if found { //evil signer has not unbonded, kicked it out
				err := strategy.NextEpochValData.PosTable.RemovePosItem(signer, currentHeight, true)
				if err != nil {
					_, found := strategy.NextEpochValData.PosTable.UnbondPosItemMap[signer]
					if !found {
						panic(fmt.Sprintf("evil signer %v cannot be found in either posItemMap or unbondedPosItemMap of NextEpochValData.PosTable. but is in the TmAddressToSignerMap", signer))
					}
				} else {
					log.Info(fmt.Sprintf("evil signer %v got unbonded because of Evidence %v", signer, e))
				}
			} else { //he should be in the unbonded map
				_, found := strategy.NextEpochValData.PosTable.UnbondPosItemMap[signer]
				if !found {
					panic(fmt.Sprintf("evil signer %v cannot be found in either posItemMap or unbondedPosItemMap of CurrEpochValData.PosTable. but is in the TmAddressToSignerMap", signer))
				}
			}
		} else {
			log.Error(fmt.Sprintf("Fail to punish address %X. Evidence %v is too long ago?", e.Validator.Address, e))
		}
	}
}
