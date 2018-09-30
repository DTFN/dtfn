package ethereum

import (
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/privval"
	"encoding/json"
)

var cdc = amino.NewCodec()

func init() {
	cryptoAmino.RegisterAmino(cdc)
}

type TxData struct {
	Pv          *privval.FilePV
	Beneficiary string
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
