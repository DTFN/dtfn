# BLS in Gelchain

As we know, Tendermint make a consensus about random number through BLS, and in
each height, Tendermint generates a random number different from other height,
at the end of height, Tendermint will put BLS random number in
`abciResponses.EndBlock`, as follow:

```
seedNum, _ := new(big.Int).SetString(tbls.GenerateSeed(block.LastSign), 16)
abciResponses.EndBlock, err = proxyAppConn.EndBlockSync(abci.RequestEndBlock{Height: block.Height, Seed: seedNum.Bytes()})
```

Gelchain gets the BLS random number through ABCI, the length of number is 256 bits,
Gelchain use murmur3 to change the length into 32 bits. after that, Gelchain put the
result in random func of golang as a seed, and generate another random number, this
new number is used to pick out the set of validators. Gelchain will do selection
repeatedly in PosTable to select multiple people randomly, the more times peer is
selected, the higher weight peer has. In each round, seven people are selected for
consensus.

```
func (posTable *PosTable) SelectItemBySeedValue(vrf []byte, len int) PosItem {
	res64 := murmur3.Sum32(vrf)
	r := rand.New(rand.NewSource(int64(res64) + int64(len)))
	value := r.Intn(posTable.PosArraySize)
	return *posTable.PosItemMap[posTable.PosArray[value]]
}
```