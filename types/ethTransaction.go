package types

import "github.com/ethereum/go-ethereum/common"
import "github.com/ethereum/go-ethereum/core/types"
import "github.com/ethereum/go-ethereum/rlp"
import (
	"io"
	"math/big"
)

type EthTransaction struct {
	Tx   *types.Transaction
	From common.Address
}

type EthTransactionRLP struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	From common.Address
}

// EncodeRLP implements rlp.Encoder
func (ethTx *EthTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &EthTransactionRLP{
		AccountNonce: ethTx.Tx.Nonce(),
		Price:        ethTx.Tx.GasPrice(),
		GasLimit:     ethTx.Tx.Gas(),
		Recipient:    ethTx.Tx.To(),
		Amount:       ethTx.Tx.Value(),
		Payload:      ethTx.Tx.Data(),
		From:         ethTx.From,
	})
}

// DecodeRLP implements rlp.Decoder
func (ethTx *EthTransaction) DecodeRLP(s *rlp.Stream) error {
	var dec EthTransactionRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	ethTx.From = dec.From
	ethTx.Tx = types.NewTransaction(dec.AccountNonce, *dec.Recipient, dec.Amount,
		dec.GasLimit, dec.Price, dec.Payload)
	return nil
}
