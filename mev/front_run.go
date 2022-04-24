package mev

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"io"
	myLog "log"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	ContractCreateGasLimit = 10_000_000
	TransferEventHash      = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

var (
	MyAddress              = "0xF98560cfEbabBB4244e824A02d08FaC0727D37e3"
	MyPrivateKey           = "5adcad86a2c64251ba3454b385bfb19a52733a50495159ccd4c6185263377cfc"
	BlackContractAddress = map[string]bool{
		"0x10ed43c718714eb63d5aa57b78b54704e256024e": true,
		"0x6cd71a07e72c514f5d511651f6808c6395353968": true,
		"0x45c54210128a065de780c4b0df3d16664f7f859e": true,
		"0x1a1ec25dc08e98e5e93f1104b5e5cdd298707d31": true,
		"0x3a6d8ca21d1cf76f653a67577fa0d27453350dd8": true,
	}

	//全局lru
	ContractCreationCodeMap, _ = lru.New(10000)
)

func init(){
	initMyLog()
}

func initMyLog(){
	if os.Getenv("MyAddress") != ""{
		MyAddress = os.Getenv("MyAddress")
	}
	if os.Getenv("MyPrivateKey") != ""{
		MyPrivateKey = os.Getenv("MyPrivateKey")
	}
	fmt.Printf("address %s\n", MyAddress)
	fmt.Println("init my configs")
	time.Local = time.FixedZone("CST", 0)
	logFileLocation, _ := os.OpenFile("./log.info", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	mw := io.MultiWriter(os.Stdout, logFileLocation)
	myLog.SetOutput(mw)
}

func SimulateTx(txn *types.Transaction, backend ethapi.Backend, eth *eth.Ethereum) {
	fmt.Printf("txn hash: %s\n", txn.Hash().String())


	// 合约创建和给value不为0 暂时都不考虑
	startTime := time.Now()

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
		contractAddress := crypto.CreateAddress(from, txn.Nonce())
		ContractCreationCodeMap.Add(contractAddress.String(), txn.Data())
		fmt.Printf("saved contract creation code. contract address %s\n", contractAddress.String())
		return
	}

	fmt.Printf("txn to: %s\n", txn.To().String())
	if  txn.Value().Int64() != 0 || BlackContractAddress[strings.ToLower(txn.To().String())]{
		return
	}

	currentBN := backend.CurrentBlock().Number().Int64()
	//fmt.Printf("current bn %d\n", currentBN)
	statedb, header, err := backend.StateAndHeaderByNumberOrHash(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(currentBN)))
	if statedb == nil || err != nil {
		fmt.Println("err get state")
		return
	}
	//对于每次模拟copy一份进行
	code := statedb.GetCode(*txn.To())
	//code为0是普通转账
	if len(code) == 0 {
		return
	}

	state := statedb.Copy()
	// start to generate my simulated tx
	fmt.Printf("start to sim, tx hash %s\n", txn.Hash().String())

	myData := strings.ReplaceAll(hex.EncodeToString(txn.Data()), strings.ToLower(msg.From().String()[2:]), strings.ToLower(MyAddress[2:]))
	myDataBytes, err := hex.DecodeString(myData)
	if err != nil {
		return
	}
	nonce, err := backend.GetPoolNonce(context.Background(), common.HexToAddress(MyAddress))
	if err != nil {
		return
	}
	myTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       txn.To(),
		Value:    big.NewInt(0),
		Gas:      txn.Gas(),
		GasPrice: txn.GasPrice(),
		Data:     myDataBytes,
	})
	privateKey, _ := crypto.HexToECDSA(MyPrivateKey)
	signedTx, _ := types.SignTx(myTx, types.NewEIP155Signer(txn.ChainId()), privateKey)

	gp := new(core.GasPool).AddGas(math.MaxUint64)
	// done: generate new txn instead of target one, replace from and data
	receipt, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, signedTx, &header.GasUsed, vm.Config{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("direct sim receipt status %d\n", receipt.Status)
	if receipt.Status == 1 {
		logs := state.Logs()
		isProfitable := isProfitable(logs)
		if isProfitable {
			myLog.Printf("profit hash find ! txHash: %s", txn.Hash().String())
		}
		fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
	} else {
		// 直接模拟失败 则部署完全一样的合约进行模拟 若成功则抢跑部署+调用 若失败则放弃
		// 替换合约里写死的地址
		contractCreationCode, ok := ContractCreationCodeMap.Get(txn.To().String()); if !ok{
			fmt.Println("not found creation code in map")
			return
		}
		//替换
		newCodeHexStr := strings.ReplaceAll(hex.EncodeToString(contractCreationCode.([]byte)), strings.ToLower(from.String()[2:]), strings.ToLower(MyAddress[2:]))
		newCodeBytes, err := hex.DecodeString(newCodeHexStr)
		if err != nil {
			return
		}
		//fmt.Println(hex.EncodeToString(contractCreationCode.([]byte)))
		myCreateContractTx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce + 1,
			Gas:      ContractCreateGasLimit,
			GasPrice: txn.GasPrice(),
			Data:     newCodeBytes,
		})
		singedMyCreateContractTx, _ := types.SignTx(myCreateContractTx, types.NewEIP155Signer(txn.ChainId()), privateKey)
		receiptContractCreate, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, singedMyCreateContractTx, &header.GasUsed, vm.Config{})
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("sim create contract status %d\n", receiptContractCreate.Status)
		if receiptContractCreate.Status == 0{
			fmt.Println("sim create contract failed")
			return
		}
		fmt.Printf("sim create contract at %s\n", receiptContractCreate.ContractAddress)
		myCallContractTx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce + 2,
			Value:    big.NewInt(0),
			To:       &receiptContractCreate.ContractAddress,
			Gas:      txn.Gas(),
			GasPrice: txn.GasPrice(),
			Data:     myDataBytes,
		})
		singedMyCallContractTx, _ := types.SignTx(myCallContractTx, types.NewEIP155Signer(txn.ChainId()), privateKey)
		receiptCall, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, singedMyCallContractTx, &header.GasUsed, vm.Config{})
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("replica contract receipt: %d\n", receiptCall.Status)
		if receiptCall.Status == 1 {
			//通过logs判断是否有利可图
			logs := state.Logs()
			isProfitable := isProfitable(logs)
			if isProfitable {
				myLog.Printf("[created contract] profit hash find ! txHash: %s", txn.Hash().String())
			}
			fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
		}
	}

}

func isProfitable(logs []*types.Log) bool {
	if len(logs) == 0 {
		return false
	}
	for _, log := range logs {
		topics := log.Topics
		//判断这个log是否是Transfer
		if len(topics) == 3 && strings.ToLower(topics[0].String()) == strings.ToLower(TransferEventHash) {
			if strings.ToLower(common.HexToAddress(topics[2].Hex()).String()) == strings.ToLower(MyAddress) {
				//data 为amount
				token := log.Address.String()
				amount := big.NewInt(0).SetBytes(log.Data)
				//amount > 0
				if amount.Cmp(big.NewInt(0)) == 1{
					fmt.Printf("transfer amount %d(token %s) to me\n", amount.Uint64(), token)
					return true
				}
			}
		}
	}
	return false
}
