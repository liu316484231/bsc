package mev

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
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
)




func init(){
	//initMyLog()
}


func SimulateExpTx(txn *types.Transaction, backend ethapi.Backend, eth *eth.Ethereum) {
	fmt.Printf("txn hash: %s\n", txn.Hash().String())
	msg, e := txn.AsMessage(types.LatestSignerForChainID(txn.ChainId()))
	if e != nil {
		fmt.Println(e.Error())
	}
	from := msg.From()
	if strings.ToLower(from.String()) == strings.ToLower(MyAddress) {
		return
	}
	// 部署合约我们需要存下creation code等待后续使用
	if txn.To() == nil {
		return
	}
	if BlackContractAddress[strings.ToLower(txn.To().String())]{
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
	gp := new(core.GasPool).AddGas(math.MaxUint64)

	// start to generate my simulated tx
	fmt.Printf("start to sim, tx hash %s\n", txn.Hash().String())

	ogReceipt, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, txn, &header.GasUsed, vm.Config{})
	if err != nil || ogReceipt.Status == 0{
		//原始请求就不能够成功
		return
	}
	logs := state.Logs()
	if len(logs) == 0 {
		return
	}
	for _, log := range logs {
		topics := log.Topics
		//判断这个log是否是Transfer
		if len(topics) == 3 && strings.ToLower(topics[0].String()) == strings.ToLower(TransferEventHash) {
			fromAddr := common.HexToAddress(topics[1].Hex())
			toAddr := common.HexToAddress(topics[2].Hex())
			if  toAddr.String() == from.String() && fromAddr.String() != from.String(){
				//data 为amount
				token := log.Address
				amount := big.NewInt(0).SetBytes(log.Data)
				//amount > 0
				if amount.Cmp(big.NewInt(0)) == 1{
					fmt.Printf("transfer amount %d(token %s) to me\n", amount.Uint64(), token)
					myLog.Printf("tx: %s, transfer amount %s(token %s) to me\n", txn.Hash().String(), amount.String(), token)
					// 需要写一个合约 讲可能增加的erc20代币 扔到合约里计算最终拿到的等同于多少BNB
					// calculate the usd value of the free token
				}
			}
		}
	}

}

