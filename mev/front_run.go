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
	ContractFRLimit = 100_000_000
	GeneralGasLimit = 22_000
	TransferEventHash      = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	WbnbAddress = "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
	UsdtAddress = "0x55d398326f99059fF775485246999027B3197955"
	BusdAddress = "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56"
	UsdcAddress = "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"

)

var (
	MyAddress              = "0xF98560cfEbabBB4244e824A02d08FaC0727D37e3"
	MyPrivateKey           = "5adcad86a2c64251ba3454b385bfb19a52733a50495159ccd4c6185263377cfc"
	//MyContractAddress      = "0xfEcce0De36743802767d4436E1658F6A66c0200a" //fixed one, already deployed on bsc-mainnet
	PriceOracleAddress     = "0xfbD61B037C325b959c0F6A7e69D8f37770C2c550" //bsc-mainnet
	//PriceOracleAddress     = "0xb6E3aE5ef1019a202B16CCAe530C07C039F58b8d" //test
	BUSDAddress            = "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56"
	BlackContractAddress = map[string]bool{
		"0x10ed43c718714eb63d5aa57b78b54704e256024e": true,
		"0x6cd71a07e72c514f5d511651f6808c6395353968": true,
		"0x45c54210128a065de780c4b0df3d16664f7f859e": true,
		"0x1a1ec25dc08e98e5e93f1104b5e5cdd298707d31": true,
		"0x3a6d8ca21d1cf76f653a67577fa0d27453350dd8": true,
		"0x0000000000004946c0e9f43f4dee607b0ef1fa1c": true, //chi token?
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
	logFileLocation, _ := os.OpenFile("./info.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	mw := io.MultiWriter(os.Stdout, logFileLocation)
	myLog.SetOutput(mw)
}

func SimulateTx(txn *types.Transaction, backend ethapi.Backend, eth *eth.Ethereum) {
	fmt.Printf("txn hash: %s\n", txn.Hash().String())
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
	if  txn.Value().Cmp(big.NewInt(0)) != 0 || BlackContractAddress[strings.ToLower(txn.To().String())]{
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
	//code := statedb.GetCode(*txn.To())
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

	// done: generate new txn instead of target one, replace from and data
	receipt, err := core.ApplyTransaction(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, signedTx, &header.GasUsed, vm.Config{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("direct sim receipt status %d\n", receipt.Status)
	if receipt.Status == 1 {
		logs := state.Logs()
		checkProfit(logs, txn)
		//if token == nil || amount == nil{
		//	return
		//}
		////// 构建price oracle请求 理论是一定能成功的
		//priceOracleABI, err := abi.JSON(strings.NewReader(mev.PriceOracleABI))
		//if err != nil {
		//	return
		//}
		//dataPacked, err := priceOracleABI.Pack("getRate", *token, common.HexToAddress(BUSDAddress), true)
		//if err != nil{
		//	fmt.Println(err.Error())
		//	return
		//}
		//oracleAddr := common.HexToAddress(PriceOracleAddress)
		//priceTx := types.NewTx(&types.LegacyTx{
		//	Nonce:    nonce + 1,
		//	To:       &oracleAddr,
		//	Gas:      txn.Gas(),
		//	GasPrice: txn.GasPrice(),
		//	Data:     dataPacked,
		//})
		//signedPriceTx, _ := types.SignTx(priceTx, types.NewEIP155Signer(txn.ChainId()), privateKey)
		//receiptPO, result, err := core.ApplyTransactionWithResult(backend.ChainConfig(), eth.BlockChain(), &header.Coinbase, gp, state, header, signedPriceTx, &header.GasUsed, vm.Config{})
		//if err != nil {
		//	myLog.Printf("get price err: %s\n", err.Error())
		//	return
		//}
		//if receiptPO.Status == 0{
		//	myLog.Printf("get price receipt failed: %d\n", receiptPO.Status)
		//	return
		//}
		//tokenPriceRate := big.NewInt(0).SetBytes(result.ReturnData)
		//if tokenPriceRate.Cmp(big.NewInt(0)) == 0{
		//	return
		//}
		//multiplier := big.NewInt(0).Mul(amount, ETHER)
		//value := big.NewInt(0).Div(multiplier, tokenPriceRate)
		//myLog.Printf("profit calculated.. txn: %s, to: %s, token: %s, value: %s\n", txn.Hash().String(), txn.To().String(), token.String(), value.String())
		//if value.Cmp(ETHER) == 1{
		//	//todo: 大于一刀才抢跑
		//}
		//fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
	} else { // 暂时先不考虑合约创建的情况
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
			logs := receiptCall.Logs
			checkProfit(logs, txn)
			fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
		}
	}

}

// need to improve todo: 暂时只返回了一个
func checkProfit(logs []*types.Log, txn *types.Transaction) (*common.Address, *big.Int){
	if len(logs) == 0 {
		return nil, nil
	}
	for _, log := range logs {
		topics := log.Topics

		//判断这个log是否是Transfer
		if len(topics) == 3 && strings.ToLower(topics[0].String()) == strings.ToLower(TransferEventHash) {
			fromAddr := common.HexToAddress(topics[1].Hex())
			toAddr := common.HexToAddress(topics[2].Hex())
			if  fromAddr.String() == toAddr.String(){
				continue
			}
			if strings.ToLower(toAddr.String()) == strings.ToLower(MyAddress){
				//data 为amount
				token := log.Address
				amount := big.NewInt(0).SetBytes(log.Data)
				if amount.Cmp(big.NewInt(0)) == 1{
					myLog.Printf("tx: %s, transfer amount %s(token %s) to me\n", txn.Hash().String(), amount.String(), token)
				}
				//amount > 0
				if strings.ToLower(token.String()) == strings.ToLower(WbnbAddress) && amount.Cmp(big.NewInt(0).Div(ETHER, big.NewInt(100))) == 1{
					myLog.Printf("tx: %s, transfer amount %s(token %s) to me\n", txn.Hash().String(), amount.String(), token)
					//front run
				}
				if (strings.ToLower(token.String()) == strings.ToLower(BusdAddress) || strings.ToLower(token.String()) == strings.ToLower(UsdcAddress) || strings.ToLower(token.String()) == strings.ToLower(UsdtAddress)) && amount.Cmp(ETHER) == 1{
					fmt.Printf("transfer amount %d(token %s) to me\n", amount.Uint64(), token)
					myLog.Printf("tx: %s, transfer amount %s(token %s) to me\n", txn.Hash().String(), amount.String(), token)
					// 需要写一个合约 将可能增加的erc20代币 扔到合约里计算最终拿到的等同于多少BNB
					// calculate the usd value of the free token
					return &token, amount
				}
			}
		}
	}
	return nil, nil
}
