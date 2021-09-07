package app

import (
	"fmt"
	emtTypes "github.com/DTFN/dtfn/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txfilter"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

// validateTx checks the validity of a tx against the blockchain's current state.
// it duplicates the logic in ethereum's tx_pool
func (app *EthermintApplication) validateTxWithFrom(tx *ethTypes.Transaction, checkType abciTypes.CheckTxType,
	fromTm common.Address, relayerTm common.Address, isRelayTxTm bool) abciTypes.ResponseCheckTx {
	// Heuristic limit, reject transactions over 32KB to prevent DOS attacks
	if tx.Size() > maxTransactionSize {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInternal),
			Log:  core.ErrOversizedData.Error()}
	}

	var signer ethTypes.Signer = ethTypes.HomesteadSigner{}
	if tx.Protected() {
		signer = app.strategy.Signer()
	}


	var from, relayer common.Address
	var txInfo ethTypes.TxInfo
	var cached bool
	success := false
	isRelayTx := false
	txHash := tx.Hash()
	if checkType == abciTypes.CheckTxType_Recheck {
		txInfo, cached = app.backend.FetchCachedTxInfo(txHash)
		if !cached {
			panic(fmt.Sprintf("The from address of tx should stay in cached"))
		} else {
			defer func() {
				if !success {
					app.logger.Error("clear cached Tx Info")
					app.backend.DeleteCachedTxInfo(txHash)
				}
			}()
		}
		from = txInfo.From
		isRelayTx = txInfo.SubTx != nil
		if isRelayTx {
			relayer = txInfo.RelayFrom
		}
	} else {
		var subTx *ethTypes.Transaction
		if tx.To() != nil {
			if txfilter.IsRelayTxFromClient(*tx.To()) {
				var err error
				// Make sure the transaction is signed properly
				from =  fromTm
				subTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx decode failed. %v",
							core.ErrInvalidSender.Error())}
				}
				relayer = relayerTm
				err = ethTypes.CheckRelayerTx(tx, subTx)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Relayer tx not match with main tx, please check, %v", err)}
				}
				isRelayTx = true
			} else if txfilter.IsRelayTxFromRelayer(*tx.To()) {
				var err error
				subTx, err = ethTypes.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInternal),
						Log: fmt.Sprintf("relayer sub tx decode failed. %v",
							core.ErrInvalidSender.Error())}
				}
				err = ethTypes.CheckRelayerTx(tx, subTx)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Relayer tx not match with main tx, please check, %v", err)}
				}
				from = fromTm
				tx.SetFrom(signer, from)
				relayer = relayerTm
				isRelayTx = true
			} else {
				// Make sure the transaction is signed properly
				from = fromTm
			}
		} else {
			// Make sure the transaction is signed properly
			from = fromTm
		}
		txInfo = ethTypes.TxInfo{Tx: tx, From: from, SubTx: subTx, RelayFrom: relayer}
	}

	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInvalidCoins),
			Log:  core.ErrNegativeValue.Error()}
	}

	currentState := app.checkTxState

	// Make sure the account exist - cant send from non-existing account.
	if checkType != abciTypes.CheckTxType_Local && !currentState.Exist(from) {
		app.logger.Info(fmt.Sprintf("receive a remote tx with not existed from %X", from))
		/*return abciTypes.ResponseCheckTx{
		Code: uint32(emtTypes.CodeUnknownAddress),
		Log:  core.ErrInvalidSender.Error()}*/
	}

	// Check the transaction doesn't exceed the current block limit gas.
	gasLimit := app.backend.GasLimit()
	if gasLimit < tx.Gas() {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInternal),
			Log:  core.ErrGasLimitReached.Error()}
	}

	// Check if nonce is not strictly increasing
	nonce := currentState.GetNonce(from)
	if nonce != tx.Nonce() {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInvalidSequence),
			Log: fmt.Sprintf(
				"Nonce for %X not strictly increasing. Expected %d Got %d .",
				from, nonce, tx.Nonce())}
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	var currentBalance *big.Int
	if isRelayTx {
		currentBalance = currentState.GetBalance(relayer)
		fmt.Printf("checkTx, using relayer %X balance \n", relayer)
	} else {
		currentBalance = currentState.GetBalance(from)
	}

	if currentBalance.Cmp(tx.Cost()) < 0 {
		return abciTypes.ResponseCheckTx{
			// TODO: Add emtTypes.CodeTypeInsufficientFunds ?
			Code: uint32(emtTypes.CodeInsufficientFunds),
			Log: fmt.Sprintf(
				"Current balance: %s, tx cost: %s",
				currentBalance, tx.Cost())}
	}

	intrGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true, false) // homestead == true

	if err != nil && tx.Gas() < intrGas {
		return abciTypes.ResponseCheckTx{
			Code: uint32(emtTypes.CodeInsufficientCoins),
			Log:  err.Error()}
	}
	height := app.backend.Es().WorkState().Height()
	err = txfilter.IsBetBlocked(from, tx.To(), currentBalance, tx.Data(), height, false)
	if err != nil {
		return abciTypes.ResponseCheckTx{
			// TODO: Add emtTypes.CodeTypeTxIsBlocked ?
			Code: uint32(emtTypes.CodeInvalidAddress),
			Log: fmt.Sprintf(
				"Tx is blocked: %v",
				err)}
	}

	if tx.To() != nil {
		if txfilter.IsAuthTx(*tx.To()) {
			err := txfilter.IsAuthBlocked(from, tx.Data(), height, false)
			if err != nil {
				return abciTypes.ResponseCheckTx{
					Code: uint32(emtTypes.CodeInvalidSequence),
					Log: fmt.Sprintf(
						"Auth tx failed, %v", err)}
			}
			currentState.SubBalance(from, tx.Cost())
		} else {
			if isRelayTx {
				currentState.SubBalance(relayer, tx.Cost())
			} else if txfilter.IsMintTx(*tx.To()) {
				err := txfilter.IsMintBlocked(from)
				if err != nil {
					return abciTypes.ResponseCheckTx{
						Code: uint32(emtTypes.CodeInvalidSequence),
						Log: fmt.Sprintf(
							"Mint tx failed, %v", err)}
				}
				currentState.SubBalance(from, tx.Cost())
			}
		}
	} else {
		currentState.SubBalance(from, tx.Cost())
	}
	// Update ether balances
	// amount + gasprice * gaslimit

	// tx.To() returns a pointer to a common address. It returns nil
	// if it is a contract creation transaction.
	if to := tx.To(); to != nil && !txfilter.IsMintTx(*tx.To()) {
		currentState.AddBalance(*to, tx.Value())
	}
	currentState.SetNonce(from, tx.Nonce()+1)

	if !cached {
		app.backend.InsertCachedTxInfo(txHash, txInfo)
	}
	success = true
	return abciTypes.ResponseCheckTx{Code: abciTypes.CodeTypeOK, GasWanted: int64(intrGas)}
}

