package mev

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	myLog "log"
	"math/big"
	"strings"
	"time"
)

const (
	PancakeRouter    = "0x10ED43C718714eb63d5aA57B78B54704E256024E"
	HelenaSwapRouter = "0xB099ED146fAD4d0dAA31E3810591FC0554aF62bB"
	PairAddress = "0xa4AA652230c3619e12a10867F56b6b03bB3De6aD"
	WBNBAddress = "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
)

var(
	Big10 = big.NewInt(10)
	Big18 = big.NewInt(18)
	ETHER = big.NewInt(1).Exp(Big10, Big18, nil)
)

func TrackHelenaSwap(txn *types.Transaction, backend ethapi.Backend, eth *eth.Ethereum) {
	startTime := time.Now()
	if txn == nil || txn.To() == nil{
	//|| strings.ToLower(txn.To().String()) != strings.ToLower(PancakeRouter) || strings.ToLower(txn.To().String()) != strings.ToLower(HelenaSwapRouter) {
		return
	}
	fmt.Printf("hash: %s, txn to: %s\n", txn.Hash().String(), txn.To().String())

	currentBN := backend.CurrentBlock().Number().Int64()
	//fmt.Printf("current bn %d\n", currentBN)
	statedb, header, err := backend.StateAndHeaderByNumberOrHash(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(currentBN)))
	if statedb == nil || err != nil {
		fmt.Println("err get state")
		return
	}
	//对于每次模拟copy一份进行
	state := statedb.Copy()
	// start to generate my simulated tx
	fmt.Printf("start to sim, tx hash %s\n", txn.Hash().String())
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	// done: generate new txn instead of target one, replace from and data
	receipt, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, txn, &header.GasUsed, vm.Config{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("sim receipt status %d\n", receipt.Status)

	if receipt.Status == 1 {
		logs := state.Logs()
		parseLogs(logs, txn, backend)
		fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
	}
}

func parseLogs(logs []*types.Log, txn *types.Transaction, backend ethapi.Backend) {
	if len(logs) == 0 {
		return
	}
	for _, log := range logs {
		topics := log.Topics
		//判断这个log是否是Transfer
		if len(topics) == 3 && strings.ToLower(topics[0].String()) == strings.ToLower(TransferEventHash) {
			//data 为amount
			token := log.Address.String()
			//from := common.HexToAddress(topics[1].Hex()).String()
			//to := common.HexToAddress(topics[2].Hex()).String()
			amount := big.NewInt(0).SetBytes(log.Data)
			if strings.ToLower(token) == strings.ToLower(WBNBAddress){
				//&& strings.ToLower(from) == strings.ToLower(PairAddress)
				// 卖HELENA
				if amount.Cmp(ETHER) == 1{
					myLog.Printf("sell hash: %s", txn.Hash().String())

				}

			}

			if  strings.ToLower(token) == strings.ToLower(WBNBAddress){
				//&& to == PairAddress
				// 买HELENA
				if amount.Cmp(ETHER) == 1{
					myLog.Printf("sell hash: %s", txn.Hash().String())

				}
			}

		}
	}
}
