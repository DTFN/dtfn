package types

import "github.com/ethereum/go-ethereum/common"
import "github.com/ethereum/go-ethereum/core/types"
import "github.com/ethereum/go-ethereum/rlp"
import (
	"io"
	"bytes"
)

type EthTransaction struct {
	Tx   *types.Transaction
	From common.Address
}

type EthTransactionRLP struct {
	TxData []byte
	From   common.Address
}

// EncodeRLP implements rlp.Encoder
func (ethTx *EthTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &EthTransactionRLP{ethTx.Tx.Data(), ethTx.From})
}

// DecodeRLP implements rlp.Decoder
func (ethTx *EthTransaction) DecodeRLP(s *rlp.Stream) error {
	var dec EthTransactionRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	ethTx.From = dec.From
	tx := new(types.Transaction)
	rlpStream := rlp.NewStream(bytes.NewBuffer(dec.TxData), 0)
	tx.DecodeRLP(rlpStream)
	ethTx.Tx = tx
	return nil
}
