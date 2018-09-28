package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
	abciTypes "github.com/tendermint/tendermint/abci/types"
)

func TestUpsertPosTable(t *testing.T) {
	pubk:=abciTypes.PubKey{}
	table := NewPosTable(1000)
	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 300,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)

	require.Equal(t, 0, table.PosArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 1000,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 1, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 3500,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 3, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 6500,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 9, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 500,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 9, table.PosArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)
}

func TestRemovePosTable(t *testing.T) {
	pubk:=abciTypes.PubKey{}
	table := NewPosTable(1000)
	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 300,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)

	require.Equal(t, 0, table.PosArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 1000,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 3500,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 6500,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 500,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)

	table.RemovePosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"))
	require.Equal(t, 3, table.PosArraySize)

	table.RemovePosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"))
	require.Equal(t, 0, table.PosArraySize)
}

func TestComplicated(t *testing.T) {
	pubk:=abciTypes.PubKey{}
	table := NewPosTable(1000)

	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 1000,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 1, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[0].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 3200,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 4, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[1].Signer)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[2].Signer)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[3].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 3610,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 6, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[4].Signer)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[5].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), 2116,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 8, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.PosArray[6].Signer)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.PosArray[7].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 4610,common.HexToAddress("0x0000000000000000000000000000000000000001"),pubk)
	require.Equal(t, 9, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[0].Signer)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[4].Signer)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[5].Signer)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[8].Signer)
	require.NoError(t, err)

	removeFlag, error := table.RemovePosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"))
	require.Equal(t, true, removeFlag)
	require.Equal(t, 5, table.PosArraySize)
	require.NoError(t, error)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.PosArray[0].Signer)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.PosArray[4].Signer)

	removeFlag, error = table.RemovePosItem(common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"))
	require.Equal(t, true, removeFlag)
	require.Equal(t, 3, table.PosArraySize)
	require.NoError(t, error)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[0].Signer)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[1].Signer)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[2].Signer)
}

