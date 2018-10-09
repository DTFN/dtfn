package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"testing"
)

func TestUpsertPosTable(t *testing.T) {
	pubk := abciTypes.PubKey{}
	table := NewPosTable(big.NewInt(1000))
	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(300), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)

	require.Equal(t, 0, table.PosArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(1000), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 1, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(3500), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 3, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), big.NewInt(6500), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 9, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), big.NewInt(500), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 9, table.PosArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)
}

func TestRemovePosTable(t *testing.T) {
	pubk := abciTypes.PubKey{}
	table := NewPosTable(big.NewInt(1000))
	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(300), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)

	require.Equal(t, 0, table.PosArraySize)
	require.Equal(t, false, upsertFlag)
	require.Error(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(1000), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(3500), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), big.NewInt(6500), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), big.NewInt(500), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)

	table.RemovePosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"))
	require.Equal(t, 3, table.PosArraySize)

	table.RemovePosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"))
	require.Equal(t, 0, table.PosArraySize)
}

func TestComplicated(t *testing.T) {
	pubk := abciTypes.PubKey{}
	table := NewPosTable(big.NewInt(1000))

	upsertFlag, err := table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(1000), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 1, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[0].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), big.NewInt(3200), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 4, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[1].Signer)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[2].Signer)
	require.Equal(t, common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), table.PosArray[3].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(3610), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 6, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[4].Signer)
	require.Equal(t, common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), table.PosArray[5].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), big.NewInt(2116), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
	require.Equal(t, 8, table.PosArraySize)
	require.Equal(t, true, upsertFlag)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.PosArray[6].Signer)
	require.Equal(t, common.HexToAddress("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"), table.PosArray[7].Signer)
	require.NoError(t, err)

	upsertFlag, err = table.UpsertPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(4610), common.HexToAddress("0x0000000000000000000000000000000000000001"), pubk)
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

func TestSelectItemByRandomValue(t *testing.T) {
	pubk := abciTypes.PubKey{}
	table := NewPosTable(big.NewInt(1000))
	table.PosArray[0] = newPosItem(common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), big.NewInt(500), common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), pubk)
	table.PosArray[1] = newPosItem(common.HexToAddress("0xa62142888aba8370742be823c1782d17a0389da1"), big.NewInt(1500), common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d"), pubk)
	table.PosArraySize = 2
	for height := 200; height <= 210; height++ {
		testItem := table.SelectItemByRandomValue(height)
		// 根据SelectItemByRandomValue逻辑，我们已经设定PosArray的具体长度为2,内部元素为table.PosArray[0]与[1]
		// 所以随机选取时,肯定在table.PosArray[0]与[1]中选,那么Balance的值不是500就是1500
		if testItem.Signer == common.HexToAddress("0xe41bf6b389b9007a3436ea1de3257583241ebe3d") {
			require.Equal(t, big.NewInt(500), testItem.Balance)
		} else {
			require.Equal(t, big.NewInt(1500), testItem.Balance)
		}
	}
}
