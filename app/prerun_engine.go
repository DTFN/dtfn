package app

import (
	"encoding/hex"
	"fmt"
	emtTypes "github.com/DTFN/dtfn/types"
	"github.com/DTFN/dtfn/version"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txfilter"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"strings"
	"sync/atomic"
)

func (app *EthermintApplication) StartPreExecuteEngine() {
	for obj := range app.preExecutedTx {
		resetFlag := atomic.LoadInt32(&app.atomResetingFlag)
		if resetFlag == 1 {
			app.logger.Info("reset block come, we need to discard it.")
			app.preExecutedNumsTx--
			if app.preExecutedNumsTx == 0 {
				app.preExecutedUsed <- 1
			}
		} else {
			app.preExecuteTx(obj)
			app.preExecutedNumsTx--
			if app.preExecutedNumsTx == 0 {
				app.logger.Info("executed pre tx completed")
				app.preExecutedUsed <- 1
			}
		}

	}
}

func (app *EthermintApplication) InsertTxIntoChannel(temp []byte) {
	app.preExecutedTx <- temp
}

func (app *EthermintApplication) PreBeginBlock(beginBlock abciTypes.RequestBeginBlock) {
	app.strategy.NextEpochValData.PosTable.ChangedFlagThisBlock = false
	header := beginBlock.GetHeader()

	app.logger.Info("begin block", "tx size", header.NumTxs)
	app.preExecutedNumsTx = header.NumTxs
	if app.preExecutedNumsTx == 0 {
		app.preExecutedUsed <- 1
	}

	// update the eth header with the tendermint header!breaking!!
	app.backend.UpdateHeaderWithTimeInfo(&header)
	app.strategy.HFExpectedData.Height = beginBlock.GetHeader().Height
	app.strategy.HFExpectedData.BlockVersion = beginBlock.GetHeader().Version.App
	app.strategy.CurrentHeightValData.Height = beginBlock.GetHeader().Height
	//when we reach the upgrade height,we change the blockversion

	if app.strategy.HFExpectedData.IsHarfForkPassed {
		for i := len(version.HeightArray) - 1; i >= 0; i-- {
			if app.strategy.HFExpectedData.Height >= version.HeightArray[i] {
				app.strategy.HFExpectedData.BlockVersion = uint64(version.VersionArray[i])
				break
			}
		}
	}
	app.logger.Info("block version", "appVersion", app.strategy.HFExpectedData.BlockVersion)
	txfilter.AppVersion = app.strategy.HFExpectedData.BlockVersion
	//if app.strategy.HFExpectedData.IsHarfForkPassed && app.strategy.HFExpectedData.Height == version.NextHardForkHeight {
	//	app.strategy.HFExpectedData.BlockVersion = version.NextHardForkVersion
	//}

	app.strategy.CurrentHeightValData.ProposerAddress = strings.ToUpper(hex.EncodeToString(beginBlock.Header.ProposerAddress))
	coinbase := app.Receiver()
	app.backend.Es().UpdateHeaderCoinbase(coinbase)
	app.strategy.CurrentHeightValData.LastVoteInfo = beginBlock.LastCommitInfo.Votes

	app.preTempBeginBlock = beginBlock
}

func (app *EthermintApplication) preExecuteTx(txBytes []byte) abciTypes.ResponseDeliverTx {
	tx, err := decodeTx(txBytes)

	if err != nil {
		// nolint: errcheck
		app.logger.Debug("DelivexTx: Received invalid transaction", "tx", tx, "err", err)
		return abciTypes.ResponseDeliverTx{
			Code: uint32(emtTypes.CodeInternal),
			Log:  err.Error(),
		}
	}

	txHash := tx.Hash()
	txInfo, ok := app.backend.FetchCachedTxInfo(txHash)
	if !ok {
		var signer ethTypes.Signer = ethTypes.HomesteadSigner{}
		if tx.Protected() {
			signer = app.strategy.Signer()
		}
		if tx.To() != nil {
			if txfilter.IsRelayTxFromRelayer(*tx.To()) {
				txInfo = ethTypes.TxInfo{Tx: tx}
				txInfo.SubTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx decode failed. %v",
							core.ErrInvalidSender.Error())}
				}
				err = ethTypes.CheckRelayerTx(tx, txInfo.SubTx)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Relayer tx not match with main tx, please check, %v", err)}
				}
				txForVerify, err := txInfo.SubTx.WithVRS(tx.RawSignatureValues())
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx WithVRS failed. %v",
							core.ErrInvalidSender.Error())}
				}
				txInfo.From, err = ethTypes.Sender(signer, txForVerify)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
				tx.SetFrom(signer, txInfo.From)
				txInfo.RelayFrom, err = ethTypes.DeriveRelayer(txInfo.From, txInfo.SubTx)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer signature verified failed. %v",
							core.ErrInvalidSender.Error())}
				}
			} else {
				// Make sure the transaction is signed properly
				from, err := ethTypes.Sender(signer, tx)
				if err != nil {
					return abciTypes.ResponseDeliverTx{
						Code: uint32(emtTypes.CodeInternal),
						Log:  core.ErrInvalidSender.Error()}
				}
				txInfo = ethTypes.TxInfo{Tx: tx, From: from}

				if txfilter.IsRelayTxFromClient(*tx.To()) {
					txInfo.SubTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
					if err != nil {
						return abciTypes.ResponseDeliverTx{
							Code: uint32(emtTypes.CodeInternal),
							Log: fmt.Sprintf("relayer sub tx decode failed. %v",
								core.ErrInvalidSender.Error())}
					}
					err = ethTypes.CheckRelayerTx(tx, txInfo.SubTx)
					if err != nil {
						return abciTypes.ResponseDeliverTx{
							Code: uint32(emtTypes.CodeInvalidSequence),
							Log: fmt.Sprintf(
								"Relayer tx not match with main tx, please check, %v", err)}
					}
					txInfo.RelayFrom, err = ethTypes.DeriveRelayer(from, txInfo.SubTx)
					if err != nil {
						return abciTypes.ResponseDeliverTx{
							Code: uint32(emtTypes.CodeInternal),
							Log: fmt.Sprintf("relayer signature verified failed. %v",
								core.ErrInvalidSender.Error())}
					}
				}
			}
		} else {
			// Make sure the transaction is signed properly
			from, err := ethTypes.Sender(signer, tx)
			if err != nil {
				return abciTypes.ResponseDeliverTx{
					Code: uint32(emtTypes.CodeInternal),
					Log:  core.ErrInvalidSender.Error()}
			}
			txInfo = ethTypes.TxInfo{Tx: tx, From: from}
		}
	}

	// we needn't delete it. because we don't have it.
	//} else {
	//	app.backend.DeleteCachedTxInfo(txHash)
	//}

	res := app.backend.DeliverTx(tx, app.strategy.HFExpectedData.BlockVersion, txInfo)
	if res.IsErr() {
		// nolint: errcheck
		app.logger.Error("DeliverTx: Error delivering tx to ethereum backend", "tx", tx,
			"err", err)
		return res
	}

	app.backend.InsertNeedClearTxHash(txHash, app.backend.Ethereum().BlockChain().CurrentBlock().Number().Int64())

	//app.CollectTx(tx)
	return abciTypes.ResponseDeliverTx{
		Code: abciTypes.CodeTypeOK,
	}
}
