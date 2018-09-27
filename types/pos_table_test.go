package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpsertPosTable(t *testing.T) {
	table := NewPosTable(1000)
	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 300,nil,nil)

	require.Equal(t, 0, table.posArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 1000,nil,nil)
	require.Equal(t, 1, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 3500,nil,nil)
	require.Equal(t, 3, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 6500,nil,nil)
	require.Equal(t, 9, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 500,nil,nil)
	require.Equal(t, 9, table.posArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)
}

func TestRemovePosTable(t *testing.T) {
	table := NewPosTable(1000)
	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 300,nil,nil)

	require.Equal(t, 0, table.posArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 1000,nil,nil)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 3500,nil,nil)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 6500,nil,nil)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 500,nil,nil)

	table.RemovePosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"))
	require.Equal(t, 3, table.posArraySize)

	table.RemovePosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"))
	require.Equal(t, 0, table.posArraySize)
}

func TestComplicated(t *testing.T) {
	table := NewPosTable(1000)

	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 1000,nil,nil)
	require.Equal(t, 1, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[0].Account)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), 3200,nil,nil)
	require.Equal(t, 4, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.posArray[1].Account)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.posArray[2].Account)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.posArray[3].Account)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 3610,nil,nil)
	require.Equal(t, 6, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[4].Account)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[5].Account)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), 2116,nil,nil)
	require.Equal(t, 8, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.posArray[6].Account)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.posArray[7].Account)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), 4610,nil,nil)
	require.Equal(t, 9, table.posArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[0].Account)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[4].Account)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[5].Account)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.posArray[8].Account)
	require.NoError(t, err)

	removeFlag, error := table.RemovePosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"))
	require.Equal(t, true, removeFlag)
	require.Equal(t, 5, table.posArraySize)
	require.NoError(t, error)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.posArray[0].Account)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.posArray[4].Account)

	removeFlag, error = table.RemovePosItem(common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"))
	require.Equal(t, true, removeFlag)
	require.Equal(t, 3, table.posArraySize)
	require.NoError(t, error)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.posArray[0].Account)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.posArray[1].Account)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.posArray[2].Account)
}

