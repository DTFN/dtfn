package ethereum

import (
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	emtTypes "github.com/green-element-chain/gelchain/types"
	rpcClient "github.com/tendermint/tendermint/rpc/lib/client"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
)

//----------------------------------------------------------------------
// Transactions sent via the go-ethereum rpc need to be routed to tendermint

// listen for txs and forward to tendermint
func (b *Backend) txBroadcastLoop() {
	//b.txSub = b.ethereum.EventMux().Subscribe(core.TxPreEvent{})
	ch := make(chan core.TxPreEvent, 100)
	sub := b.ethereum.TxPool().SubscribeTxPreEvent(ch)
	defer close(ch)
	defer sub.Unsubscribe()

	waitForServer(b.client)

	//for obj := range b.txSub.Chan() {
	for obj := range ch {
		if err := b.BroadcastTx(&emtTypes.EthTransaction{obj.Tx,obj.From}); err != nil {
			log.Error("Broadcast error", "err", err)
			obj.Result <- err
			go b.ethereum.TxPool().RemoveTx(obj.Tx.Hash()) //start a goroutine to avoid deadlock
		} else {
			obj.Result <- nil
		}
	}
}

func (b *Backend) BroadcastTxSync(tx tmTypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	resCh := make(chan *abciTypes.Response, 1)
	err := b.memPool.CheckTx(tx, func(res *abciTypes.Response) {
		resCh <- res
	})
	if err != nil {
		return nil, fmt.Errorf("Error broadcasting transaction: %v", err)
	}
	res := <-resCh
	r := res.GetCheckTx()
	return &ctypes.ResultBroadcastTx{
		Code: r.Code,
		Data: r.Data,
		Log:  r.Log,
		Hash: tx.Hash(),
	}, nil
}

// BroadcastTx broadcasts a transaction to tendermint core
// #unstable
func (b *Backend) BroadcastTx(tx *emtTypes.EthTransaction) error {

	txBytes, err := rlp.EncodeToBytes(tx)
	if err != nil {
		log.Error("tx %v EncodeToBytes err %v", tx, err)
		return err
	}

	/*	buf := new(bytes.Buffer)
		if err := tx.EncodeRLP(buf); err != nil {
			return err
		}*/
	/*	params := map[string]interface{}{
			"tx": buf.Bytes(),
		}*/
	tmTx := tmTypes.Tx(txBytes)
	result, err := b.BroadcastTxSync(tmTx)
	//result, err := b.client.Call("broadcast_tx_sync", params, &result)
	if err != nil {
		log.Error("broadcast_tx_sync return err %v", err)
		return err
	}
	if result.Code != abciTypes.CodeTypeOK {
		err = fmt.Errorf("Error on broadcast_tx_sync. result: %v", result)
		return err
	}
	return nil
}

//----------------------------------------------------------------------
// wait for Tendermint to open the socket and run http endpoint

func waitForServer(c rpcClient.HTTPClient) {
	ctypes.RegisterAmino(c.Codec())
	result := new(ctypes.ResultStatus)

	for {
		_, err := c.Call("status", map[string]interface{}{}, result)
		if err == nil {
			break
		}

		log.Info("Waiting for tendermint endpoint to start", "err", err)
		time.Sleep(time.Second * 3)
	}
}
