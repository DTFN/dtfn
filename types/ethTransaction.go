package types

import "github.com/ethereum/go-ethereum/common"
import "github.com/ethereum/go-ethereum/core/types"
import "github.com/ethereum/go-ethereum/rlp"
import "io"

type EthTransaction struct {
	Tx   *types.Transaction
	From common.Address
}

// EncodeRLP implements rlp.Encoder
func (tx *EthTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, tx)
}

// DecodeRLP implements rlp.Decoder
func (tx *EthTransaction) DecodeRLP(s *rlp.Stream) error {
	err := s.Decode(tx)
	return err
}
