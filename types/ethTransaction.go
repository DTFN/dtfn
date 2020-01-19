package types

import "github.com/ethereum/go-ethereum/common"
import "github.com/ethereum/go-ethereum/core/types"


type TxInfo struct {
	From    common.Address
	SubTx   *types.Transaction
	SubFrom common.Address
}
