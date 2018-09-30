package ethereum

import (
	"testing"
	"fmt"
)

var input = `{
  "pub_key":{
    "type":"tendermint/PubKeyEd25519",
    "value":"q/7QL3skC/rvTYRXOO9I5y+RWOhahr9WjyNHkcf8OQ8="
  },
  "beneficiary":"0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"
}`

func TestMarshalTxData(t *testing.T) {
	data, _ := MarshalTxData(input)
	fmt.Println(data.Pv.PubKey)
	fmt.Println(data.Beneficiary)
}
