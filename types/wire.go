package types

import (
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/privval"
)

var cdc = amino.NewCodec()

func init() {
	RegisterBlockAmino(cdc)
}

func RegisterBlockAmino(cdc *amino.Codec) {
	cryptoAmino.RegisterAmino(cdc)
}


func MarshalPubKey(jsonStr string) (crypto.PubKey, error) {
	jsonByte := []byte(jsonStr)
	pv := &privval.FilePV{}
	err := cdc.UnmarshalJSON(jsonByte, &pv)
	if err != nil {
		return nil, err
	} else {
		return pv.PubKey, nil
	}
}
