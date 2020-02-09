package ethereum

import (
	"github.com/green-element-chain/gelchain/types"
	"time"

	"fmt"
	"github.com/ethereum/go-ethereum/core"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpcClient "github.com/tendermint/tendermint/rpc/lib/client"
	tmTypes "github.com/tendermint/tendermint/types"
)

//----------------------------------------------------------------------
// Transactions sent via the go-ethereum rpc need to be routed to tendermint

// listen for txs and forward to tendermint
func (b *Backend) txBroadcastLoop() {
	//b.txSub = b.ethereum.EventMux().Subscribe(core.TxPreEvent{})
	ch := make(chan core.TxPreEvent, 50000)
	sub := b.ethereum.TxPool().SubscribeTxPreEvent(ch)
	defer close(ch)
	defer sub.Unsubscribe()

	waitForServer(b.client)
	b.ethereum.TxPool().BeginConsume()
	//for obj := range b.txSub.Chan() {
	txCount := 0
	for obj := range ch {
		fmt.Println("----------------we are receive txpreevent from txpool------------------")
		fmt.Println(obj.From.String())
		fmt.Println("----------------we are receive txpreevent from txpool------------------")

		b.txInfo = types.TxInfo{
			From:      obj.From,
			IsRelayTx: obj.RelayTxFlag,
			RelayFrom: obj.RelayAddress,
		}
		fmt.Println("---------------TxInfo-------------------")
		fmt.Println(b.txInfo)
		fmt.Println("---------------TxInfo-------------------")

		if err := b.BroadcastTx(obj.Tx); err != nil {
			log.Error("Broadcast error", "err", err)
			obj.Result <- err
			go b.ethereum.TxPool().RemoveTx(obj.Tx.Hash()) //start a goroutine to avoid deadlock
		} else {
			obj.Result <- nil
		}
		if txCount > 1<<10 {
			b.ethereum.TxPool().SetFlowLimit(b.memPool.Size())
			txCount = 0
		}
		txCount++
	}
}

func (b *Backend) BroadcastTxSync(tx tmTypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	resCh := make(chan *abciTypes.Response, 1)
	err := b.memPool.CheckTxLocal(tx, func(res *abciTypes.Response) {
		resCh <- res
	})
	if err != nil {
		return nil, fmt.Errorf("Error broadcasting transaction: %v ", err)
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
func (b *Backend) BroadcastTx(tx *ethTypes.Transaction) error {
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
		return err
	}
	if result.Code != abciTypes.CodeTypeOK {
		return fmt.Errorf("CheckTx fail. result: %v ", result)
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
