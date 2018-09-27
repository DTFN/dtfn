package types

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMarshalPubkey(t *testing.T) {
	jsonStr := "{  \"pub_key\": {   \"type\": \"tendermint/PubKeyEd25519\",   \"value\": \"q/7QL3skC/rvTYRXOO9I5y+RWOhahr9WjyNHkcf8OQ8=\"  } }"
	pubKey, err := marshalPubKey(jsonStr)
	require.NoError(t, err)
	require.Equal(t, strings.ToLower("D7891977473805B1F3B1B90BA6AE0EFD999B75DC"), hex.
		EncodeToString(pubKey.Address()))
}
