package mev

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	mev "github.com/ethereum/go-ethereum/mev/abi"
	"github.com/ethereum/go-ethereum/rpc"
	myLog "log"
	"math/big"
	"strings"
	"time"
)

var(
	Big10 = big.NewInt(10)
	Big9 = big.NewInt(9)
	Big18 = big.NewInt(18)
	ETHER = big.NewInt(1).Exp(Big10, Big18, nil)
	GWEI = big.NewInt(1).Exp(Big10, Big9, nil)
)

const(
	skimTokenContractAddress = "0x03275A7751D610bd4350e884fA90e0D6e64470DC"
)

func SimRebaseTokenSkim(backend ethapi.Backend, eth *eth.Ethereum, maxGas *big.Int) {
	startTime := time.Now()
	// 构造请求
	ABI, err := abi.JSON(strings.NewReader(mev.SkimTokenExpABI))
	if err != nil {
		return
	}
	dataPacked, err := ABI.Pack("exp")
	if err != nil{
		fmt.Println(err.Error())
		return
	}
	skimTokenContract := common.HexToAddress(skimTokenContractAddress)
	nonce, err := backend.GetPoolNonce(context.Background(), common.HexToAddress(MyAddress))
	if err != nil {
		return
	}
	priceTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &skimTokenContract,
		Gas:      5_000_000,
		GasPrice: maxGas,
		Data:     dataPacked,
	})
	privateKey, _ := crypto.HexToECDSA(MyPrivateKey)
	signedTx, _ := types.SignTx(priceTx, types.NewEIP155Signer(big.NewInt(56)), privateKey)
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
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	// done: generate new txn instead of target one, replace from and data
	receipt, result, err := core.ApplyTransactionWithResult(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, signedTx, &header.GasUsed, vm.Config{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if receipt.Status == 1 {
		// look result
		profit := big.NewInt(0).SetBytes(result.ReturnData)

		fmt.Printf("sim receipt status %d, profit: %s \n", receipt.Status, profit.String())


		if profit.Cmp(big.NewInt(0)) == 1{
			// run my tx
			myLog.Printf("found profit @ %d", currentBN)
			// backend.SendTx(context.Background(), signedTx)
		}

		fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
	}
}

