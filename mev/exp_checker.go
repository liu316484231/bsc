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
	"strings"
)


func SimulateOriginalTx(txn *types.Transaction, backend ethapi.Backend, eth *eth.Ethereum) {
	//fmt.Printf("txn hash: %s\n", txn.Hash().String())
	msg, e := txn.AsMessage(types.LatestSignerForChainID(txn.ChainId()))
	if e != nil {
		fmt.Println(e.Error())
	}
	from := msg.From()
	to := txn.To()
	//创建合约
	if to == nil{
		return
	}
	//fmt.Printf("txn to: %s\n", txn.To().String())
	if BlackContractAddress[strings.ToLower(txn.To().String())] {
		return
	}
	currentBN := backend.CurrentBlock().Number().Int64()
	//fmt.Printf("current bn %d\n", currentBN)
	statedb, header, err := backend.StateAndHeaderByNumberOrHash(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(currentBN)))
	if statedb == nil || err != nil {
		fmt.Println("err get state")
		return
	}
	//code为0是普通转账
	if len(txn.Data()) == 0 {
		return
	}
	state := statedb.Copy()
	//balanceBeforeFrom := state.GetBalance(from)
	//balanceBeforeTo := state.GetBalance(*to)
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	// start to generate my simulated tx
	fmt.Printf("start to sim, tx hash %s, from: %s \n", txn.Hash().String(), from.String())
	ogReceipt, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, txn, &header.GasUsed, vm.Config{})
	if err != nil || ogReceipt.Status == 0 {
		//原始请求就不能够成功
		return
	}
	if ogReceipt.Status == 1 {
		//原始请求成功
		//判断原生代币是否增加
		//balanceAfterFrom := state.GetBalance(from)
		//balanceAfterTo := state.GetBalance(*to)
		//if balanceAfterFrom.Sub(balanceAfterFrom, balanceBeforeFrom).Cmp(ETHER.Div(ETHER, big.NewInt(10))) == 1{
		//	myLog.Printf("tx: %s, transfer BNB to from \n", txn.Hash().String())
		//
		//}
		//if balanceAfterTo.Sub(balanceAfterTo, balanceBeforeTo).Cmp(ETHER.Div(ETHER, big.NewInt(10))) == 1{
		//	myLog.Printf("tx: %s, transfer BNB to contract \n", txn.Hash().String())
		//}
		logs := state.Logs()
		for _, log := range logs {
			topics := log.Topics
			if strings.ToLower(topics[0].String()) == strings.ToLower(DodoFlashloanEvent){
				myLog.Printf("tx: %s, dodo flashloan exploit.. \n", txn.Hash().String())
			}

			if strings.ToLower(topics[0].String()) == strings.ToLower(ValasLendingFlashloanEvent){
				myLog.Printf("tx: %s, valas lending flashloan exploit.. \n", txn.Hash().String())
			}

			//判断这个log是否是Transfer
			//if len(topics) == 3 && strings.ToLower(topics[0].String()) == strings.ToLower(TransferEventHash) {
			//	fromAddr := common.HexToAddress(topics[1].Hex())
			//	toAddr := common.HexToAddress(topics[2].Hex())
			//	if fromAddr.String() == toAddr.String() {
			//		continue
			//	}
			//	//转给from钱
			//	if strings.ToLower(toAddr.String()) == strings.ToLower(from.String()) {
			//		//data 为amount
			//		token := log.Address
			//		amount := big.NewInt(0).SetBytes(log.Data)
			//		//amount > 0
			//		if strings.ToLower(token.String()) == strings.ToLower(WbnbAddress) && amount.Cmp(big.NewInt(0).Div(ETHER, big.NewInt(10))) == 1 {
			//			myLog.Printf("tx: %s, transfer amount %s(token %s) to from\n", txn.Hash().String(), amount.String(), token)
			//		}
			//		if (strings.ToLower(token.String()) == strings.ToLower(BusdAddress) || strings.ToLower(token.String()) == strings.ToLower(UsdcAddress) || strings.ToLower(token.String()) == strings.ToLower(UsdtAddress)) && amount.Cmp(ETHER.Mul(ETHER, big.NewInt(100))) == 1 {
			//			fmt.Printf("transfer amount %d(token %s) to me\n", amount.Uint64(), token)
			//			myLog.Printf("tx: %s, transfer amount %s(token %s) to from\n", txn.Hash().String(), amount.String(), token)
			//		}
			//	}
			//	//转给合约钱
			//	if strings.ToLower(toAddr.String()) == strings.ToLower(to.String()) {
			//		//data 为amount
			//		token := log.Address
			//		amount := big.NewInt(0).SetBytes(log.Data)
			//		//amount > 0
			//		if strings.ToLower(token.String()) == strings.ToLower(WbnbAddress) && amount.Cmp(big.NewInt(0).Div(ETHER, big.NewInt(10))) == 1 {
			//			myLog.Printf("tx: %s, transfer amount %s(token %s) to from\n", txn.Hash().String(), amount.String(), token)
			//			//front run
			//		}
			//		if (strings.ToLower(token.String()) == strings.ToLower(BusdAddress) || strings.ToLower(token.String()) == strings.ToLower(UsdcAddress) || strings.ToLower(token.String()) == strings.ToLower(UsdtAddress)) && amount.Cmp(ETHER.Mul(ETHER, big.NewInt(100))) == 1 {
			//			fmt.Printf("transfer amount %d(token %s) to me\n", amount.Uint64(), token)
			//			myLog.Printf("tx: %s, transfer amount %s(token %s) to from\n", txn.Hash().String(), amount.String(), token)
			//			// 需要写一个合约 将可能增加的erc20代币 扔到合约里计算最终拿到的等同于多少BNB
			//			// calculate the usd value of the free token
			//		}
			//	}
			//}

		}

	}
}

