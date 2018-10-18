package ethereum

import (
	"encoding/json"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/privval"
)

var cdc = amino.NewCodec()

func init() {
	cryptoAmino.RegisterAmino(cdc)
}

type TxData struct {
	Pv           *privval.FilePV
	Beneficiary  string
	BlsKeyString string
}

func MarshalTxData(jsonStr string) (*TxData, error) {
	jsonByte := []byte(jsonStr)
	d := &TxData{}
	json.Unmarshal(jsonByte, d)
	pv := &privval.FilePV{}
	cdc.UnmarshalJSON(jsonByte, pv)
	d.Pv = pv
	return d, nil
}
