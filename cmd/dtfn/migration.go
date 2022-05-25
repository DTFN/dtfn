package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/DTFN/dtfn/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txfilter"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/urfave/cli.v1"
)

func makeMigrationCmd(ctx *cli.Context) error {
	// make snapshot directory and ensure it is clean
	fmt.Println("Prepare snapshot directory to store block chain state")
	snapshotDir := ctx.Args().First()
	fileInfo, err := os.Stat(snapshotDir)
	if err != nil {
		err = os.MkdirAll(snapshotDir, os.ModePerm)
		if err != nil {
			return errors.New("cannot create directory " + snapshotDir)
		}
	} else {
		if !fileInfo.IsDir() {
			return errors.New(snapshotDir + " is not a directory")
		} else {
			dir, err := ioutil.ReadDir(snapshotDir)
			if err != nil {
				return errors.New("cannot open directory " + snapshotDir)
			}
			for i, sub := range dir {
				os.RemoveAll(path.Join(snapshotDir, sub.Name()))
				// calculate progress
				percent := float64(i + 1)
				percent /= float64(len(dir))
				percent *= 100.0
				fmt.Printf("Cleaning directory %s: %.2f%%\r", snapshotDir, percent)
			}
		}
	}
	fmt.Printf("\x1b[K")
	fmt.Println("Snapshot directory is well prepared")

	// open block chain instance
	_, backend := utils.MakeMigrationNode(ctx)
	ethereumInstance := backend.Ethereum()
	if ethereumInstance == nil {
		return errors.New("cannot instantiate ethereum instance")
	}

	blockChain := ethereumInstance.BlockChain()
	if blockChain == nil {
		return errors.New("cannot get block chain")
	}

	// dump state to snapshot directory
	firstBlock := blockChain.Genesis().NumberU64()
	lastBlock := blockChain.CurrentBlock().NumberU64()
	fmt.Printf("Read blocks [%d, %d] to reconstruct preimages for contract address\n", firstBlock, lastBlock)
	preimages, txTotal, txFailed := reconstructContractPreimage(blockChain, firstBlock, lastBlock)
	fmt.Printf("\x1b[K")
	fmt.Printf("%d preimages for contract address reconstructed, %d tx in total, %d tx failed\n", len(preimages), txTotal, txFailed)

	fmt.Println("Write state to snapshot directory")
	contractWritten := writeState2File(blockChain, lastBlock, preimages, snapshotDir, txTotal, txFailed)
	fmt.Printf("\x1b[K")
	fmt.Printf("%d contracts written\n", contractWritten)
	return nil
}

// traverse through block chain to reconstruct contract address preimages
func reconstructContractPreimage(chain *core.BlockChain, firstBlock uint64, lastBlock uint64) (map[common.Hash]common.Address, uint64, uint64) {
	var txTotal uint64
	var txFailed uint64
	if chain == nil || firstBlock > lastBlock {
		return nil, txTotal, txFailed
	}

	// preimages is a map from address hash to address
	// in ethereum world state MPT, the key is not contract address, but sha3(address) instead
	preimages := make(map[common.Hash]common.Address)
	for i := firstBlock; i <= lastBlock; i++ {
		// showing progress
		percent := float64(i - firstBlock + 1)
		percent /= float64(lastBlock - firstBlock + 1)
		percent *= 100.0
		fmt.Printf("reading blocks: %.2f%%\r", percent)

		block := chain.GetBlockByNumber(i)
		if block == nil {
			continue
		}

		receipts := chain.GetReceiptsByHash(block.Hash())

		for ti, tx := range block.Transactions() {
			txTotal++
			if receipts[ti].Status != 0x01 {
				txFailed++
			}

			if tx.To() != nil && *tx.To() != txfilter.RelayTxFromClient && *tx.To() != txfilter.RelayTxFromRelayer {
				continue
			}

			isTxCreate := true
			if tx.To() != nil && (*tx.To() == txfilter.RelayTxFromClient || *tx.To() == txfilter.RelayTxFromRelayer) {
				subTransaction, err := types.DecodeTxFromHexBytes(tx.Data())
				if err != nil {
					fmt.Println("Error decode subtransaction:", err)
				}

				if subTransaction.To() != nil {
					isTxCreate = false
				}
			}

			if isTxCreate {
				h := crypto.Keccak256(receipts[ti].ContractAddress[:])
				preimages[common.BytesToHash(h)] = receipts[ti].ContractAddress
			}
		}
	}

	return preimages, txTotal, txFailed
}

// here we assume preimages have one entry for every contract address we come across
func writeState2File(chain *core.BlockChain, blockNumber uint64, preimages map[common.Hash]common.Address, targetDir string,
	txTotal uint64, txFailed uint64) uint64 {
	if chain == nil {
		return 0
	}

	// read the specified block
	block := chain.GetBlockByNumber(blockNumber)
	if block == nil {
		fmt.Println("cannot read block#", blockNumber)
		return 0
	}

	// write block number to file
	blockNumberFileName := filepath.Join(targetDir, "blockNumber")
	blockNumberFile, err := os.Create(blockNumberFileName)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	defer blockNumberFile.Close()

	blockNumberFile.WriteString(strconv.FormatUint(block.NumberU64(), 10))

	// write block hash to file
	blockHashFileName := filepath.Join(targetDir, "blockHash")
	blockHashFile, err := os.Create(blockHashFileName)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	defer blockHashFile.Close()

	blockHashFile.WriteString(block.Hash().Hex())

	// write tx summary
	txSummaryFileName := filepath.Join(targetDir, "txSummary")
	txSummaryFile, err := os.Create(txSummaryFileName)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	defer txSummaryFile.Close()

	txSummaryFile.WriteString("txTotal:")
	txSummaryFile.WriteString(strconv.FormatUint(txTotal, 10))
	txSummaryFile.WriteString("\n")
	txSummaryFile.WriteString("txFailed:")
	txSummaryFile.WriteString(strconv.FormatUint(txFailed, 10))
	txSummaryFile.WriteString("\n")

	// read state db
	stateDB, err := chain.State()
	if err != nil {
		fmt.Println("cannot open state db:", err)
		return 0
	}

	trie, err := stateDB.Database().OpenTrie(block.Root())
	if err != nil {
		fmt.Println("cannot open state trie:", err)
		return 0
	}

	iter := trie.NodeIterator(nil)
	var contractWritten uint64
	for iter.Next(true) {
		if iter.Leaf() {
			// get account
			account := state.Account{}
			err := rlp.DecodeBytes(iter.LeafBlob(), &account)
			if err != nil {
				fmt.Println("cannot get account:", err)
				continue
			}

			// get byte code
			code, err := stateDB.Database().ContractCode(common.BytesToHash(iter.LeafKey()), common.BytesToHash(account.CodeHash))
			// this might be EOA, so we cannot get its byte code
			if err != nil {
				// fmt.Printf("cannot get byte code:\naccount:%+v, err:%+v\n", account, err)
				continue
			}

			// get storage trie
			stateTrie, err := stateDB.Database().OpenStorageTrie(common.BytesToHash(iter.LeafKey()), account.Root)
			if err != nil {
				fmt.Println("cannot open storage trie:", err)
				continue
			}

			contractFileName := filepath.Join(targetDir, "contract_"+common.Bytes2Hex(iter.LeafKey()))
			if address, ok := preimages[common.BytesToHash(iter.LeafKey())]; ok {
				contractFileName = filepath.Join(targetDir, "contract_"+common.Bytes2Hex(address.Bytes()))
			} else {
				fmt.Printf("cannot get contract address %v\n", iter.LeafKey())
				continue
			}
			contractFile, err := os.Create(contractFileName)
			if err != nil {
				fmt.Println(err)
				continue
			}

			// write account to file
			contractFile.WriteString("Nonce:")
			contractFile.WriteString(strconv.FormatUint(account.Nonce, 10))
			contractFile.WriteString("\n")
			contractFile.WriteString("Balance:")
			contractFile.WriteString(account.Balance.String())
			contractFile.WriteString("\n")
			// write byte code to file
			contractFile.WriteString("CodeHash:")
			contractFile.WriteString(common.Bytes2Hex(account.CodeHash))
			contractFile.WriteString("\n")

			contractFile.WriteString("Code:")
			contractFile.WriteString(common.Bytes2Hex(code))
			contractFile.WriteString("\n")
			// write storage to file
			stateIter := stateTrie.NodeIterator(nil)
			for stateIter.Next(true) {
				if stateIter.Leaf() {
					contractFile.WriteString("Storage[")
					contractFile.WriteString(common.Bytes2Hex(stateIter.LeafKey()))
					contractFile.WriteString(",")
					var value []byte
					rlp.DecodeBytes(stateIter.LeafBlob(), &value)
					contractFile.WriteString(common.Bytes2Hex(value))
					contractFile.WriteString("]\n")
				}
			}
			contractFile.Close()
			contractWritten++

			// showing progress
			percent := float64(contractWritten)
			percent /= float64(len(preimages))
			percent *= 100.0
			fmt.Printf("writing contract state: %.2f%%\r", percent)
		}
	}

	return contractWritten
}
