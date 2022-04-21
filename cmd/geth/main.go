// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// geth is the official command-line client for Ethereum.
package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"io"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"

	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"

	"gopkg.in/urfave/cli.v1"
)
import myLog "log"
const (
	clientIdentifier = "geth" // Client identifier to advertise over the network
)

const (
	MyAddress              = "0xF98560cfEbabBB4244e824A02d08FaC0727D37e3"
	MyPrivateKey           = "5adcad86a2c64251ba3454b385bfb19a52733a50495159ccd4c6185263377cfc"
	ContractCreateGasLimit = 210000
	TransferEventHash      = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

var (
	BlackContractAddress = map[string]bool{
		"0x10ed43c718714eb63d5aa57b78b54704e256024e": true,
		"0x6cd71a07e72c514f5d511651f6808c6395353968": true,
		"0x45c54210128a065de780c4b0df3d16664f7f859e": true,
		"0x1a1ec25dc08e98e5e93f1104b5e5cdd298707d31": true,
		"0x3a6d8ca21d1cf76f653a67577fa0d27453350dd8": true,
	}
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
	// The app that holds all commands and flags.
	app = flags.NewApp(gitCommit, gitDate, "the go-ethereum command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.AncientFlag,
		utils.MinFreeDiskSpaceFlag,
		utils.KeyStoreDirFlag,
		utils.ExternalSignerFlag,
		utils.NoUSBFlag,
		utils.DirectBroadcastFlag,
		utils.DisableSnapProtocolFlag,
		utils.DiffSyncFlag,
		utils.PipeCommitFlag,
		utils.RangeLimitFlag,
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.OverrideBerlinFlag,
		utils.EthashCacheDirFlag,
		utils.EthashCachesInMemoryFlag,
		utils.EthashCachesOnDiskFlag,
		utils.EthashCachesLockMmapFlag,
		utils.EthashDatasetDirFlag,
		utils.EthashDatasetsInMemoryFlag,
		utils.EthashDatasetsOnDiskFlag,
		utils.EthashDatasetsLockMmapFlag,
		utils.TxPoolLocalsFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.TxPoolReannounceTimeFlag,
		utils.SyncModeFlag,
		utils.ExitWhenSyncedFlag,
		utils.GCModeFlag,
		utils.SnapshotFlag,
		utils.TxLookupLimitFlag,
		utils.LightServeFlag,
		utils.LightIngressFlag,
		utils.LightEgressFlag,
		utils.LightMaxPeersFlag,
		utils.LightNoPruneFlag,
		utils.LightKDFFlag,
		utils.UltraLightServersFlag,
		utils.UltraLightFractionFlag,
		utils.UltraLightOnlyAnnounceFlag,
		utils.LightNoSyncServeFlag,
		utils.WhitelistFlag,
		utils.BloomFilterSizeFlag,
		utils.TriesInMemoryFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheTrieJournalFlag,
		utils.CacheTrieRejournalFlag,
		utils.CacheGCFlag,
		utils.CacheSnapshotFlag,
		utils.CachePreimagesFlag,
		utils.PersistDiffFlag,
		utils.DiffBlockFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.MiningEnabledFlag,
		utils.MinerThreadsFlag,
		utils.MinerNotifyFlag,
		utils.MinerGasTargetFlag,
		utils.MinerGasLimitFlag,
		utils.MinerGasPriceFlag,
		utils.MinerEtherbaseFlag,
		utils.MinerExtraDataFlag,
		utils.MinerRecommitIntervalFlag,
		utils.MinerDelayLeftoverFlag,
		utils.MinerNoVerfiyFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DNSDiscoveryFlag,
		utils.MainnetFlag,
		utils.DeveloperFlag,
		utils.DeveloperPeriodFlag,
		utils.RopstenFlag,
		utils.RinkebyFlag,
		utils.GoerliFlag,
		utils.YoloV3Flag,
		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
		utils.EthStatsURLFlag,
		utils.FakePoWFlag,
		utils.NoCompactionFlag,
		utils.GpoBlocksFlag,
		utils.GpoPercentileFlag,
		utils.GpoMaxGasPriceFlag,
		utils.EWASMInterpreterFlag,
		utils.EVMInterpreterFlag,
		utils.MinerNotifyFullFlag,
		configFileFlag,
		utils.CatalystFlag,
		utils.BlockAmountReserved,
		utils.CheckSnapshotWithMPT,
	}

	rpcFlags = []cli.Flag{
		utils.HTTPEnabledFlag,
		utils.HTTPListenAddrFlag,
		utils.HTTPPortFlag,
		utils.HTTPCORSDomainFlag,
		utils.HTTPVirtualHostsFlag,
		utils.LegacyRPCEnabledFlag,
		utils.LegacyRPCListenAddrFlag,
		utils.LegacyRPCPortFlag,
		utils.LegacyRPCCORSDomainFlag,
		utils.LegacyRPCVirtualHostsFlag,
		utils.LegacyRPCApiFlag,
		utils.GraphQLEnabledFlag,
		utils.GraphQLCORSDomainFlag,
		utils.GraphQLVirtualHostsFlag,
		utils.HTTPApiFlag,
		utils.HTTPPathPrefixFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.WSPathPrefixFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.InsecureUnlockAllowedFlag,
		utils.RPCGlobalGasCapFlag,
		utils.RPCGlobalTxFeeCapFlag,
		utils.AllowUnprotectedTxs,
	}

	metricsFlags = []cli.Flag{
		utils.MetricsEnabledFlag,
		utils.MetricsEnabledExpensiveFlag,
		utils.MetricsHTTPFlag,
		utils.MetricsPortFlag,
		utils.MetricsEnableInfluxDBFlag,
		utils.MetricsInfluxDBEndpointFlag,
		utils.MetricsInfluxDBDatabaseFlag,
		utils.MetricsInfluxDBUsernameFlag,
		utils.MetricsInfluxDBPasswordFlag,
		utils.MetricsInfluxDBTagsFlag,
	}
)

func init() {
	initMyLog()
	// Initialize the CLI app and start Geth
	app.Action = geth
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2020 The go-ethereum Authors and BSC Authors"
	app.Commands = []cli.Command{
		// See chaincmd.go:
		initCommand,
		initNetworkCommand,
		importCommand,
		exportCommand,
		importPreimagesCommand,
		exportPreimagesCommand,
		removedbCommand,
		dumpCommand,
		dumpGenesisCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		consoleCommand,
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		makecacheCommand,
		makedagCommand,
		versionCommand,
		versionCheckCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
		// see dbcmd.go
		dbCommand,
		// See cmd/utils/flags_legacy.go
		utils.ShowDeprecated,
		// See snapshot.go
		snapshotCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, metricsFlags...)

	app.Before = func(ctx *cli.Context) error {
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		prompt.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// prepare manipulates memory cache allowance and setups metric system.
// This function should be called before launching devp2p stack.
func prepare(ctx *cli.Context) {
	// If we're running a known preset, log it for convenience.
	switch {
	case ctx.GlobalIsSet(utils.RopstenFlag.Name):
		log.Info("Starting Geth on Ropsten testnet...")

	case ctx.GlobalIsSet(utils.RinkebyFlag.Name):
		log.Info("Starting Geth on Rinkeby testnet...")

	case ctx.GlobalIsSet(utils.GoerliFlag.Name):
		log.Info("Starting Geth on Görli testnet...")

	case ctx.GlobalIsSet(utils.YoloV3Flag.Name):
		log.Info("Starting Geth on YOLOv3 testnet...")

	case ctx.GlobalIsSet(utils.DeveloperFlag.Name):
		log.Info("Starting Geth in ephemeral dev mode...")

	case !ctx.GlobalIsSet(utils.NetworkIdFlag.Name):
		log.Info("Starting Geth on Ethereum mainnet...")
	}
	// If we're a full node on mainnet without --cache specified, bump default cache allowance
	if ctx.GlobalString(utils.SyncModeFlag.Name) != "light" && !ctx.GlobalIsSet(utils.CacheFlag.Name) && !ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
		// Make sure we're not on any supported preconfigured testnet either
		if !ctx.GlobalIsSet(utils.RopstenFlag.Name) && !ctx.GlobalIsSet(utils.RinkebyFlag.Name) && !ctx.GlobalIsSet(utils.GoerliFlag.Name) && !ctx.GlobalIsSet(utils.DeveloperFlag.Name) {
			// Nope, we're really on mainnet. Bump that cache up!
			log.Info("Bumping default cache on mainnet", "provided", ctx.GlobalInt(utils.CacheFlag.Name), "updated", 4096)
			ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(4096))
		}
	}
	// If we're running a light client on any network, drop the cache to some meaningfully low amount
	if ctx.GlobalString(utils.SyncModeFlag.Name) == "light" && !ctx.GlobalIsSet(utils.CacheFlag.Name) {
		log.Info("Dropping default light client cache", "provided", ctx.GlobalInt(utils.CacheFlag.Name), "updated", 128)
		ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(128))
	}

	// Start metrics export if enabled
	utils.SetupMetrics(ctx)

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
}

// geth is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func geth(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	prepare(ctx)
	stack, backend, eth := makeFullNodeWithEthereum(ctx)
	defer stack.Close()

	startNode(ctx, stack, backend)
	go parsePending(stack, backend, eth)
	stack.Wait()
	return nil
}

func initMyLog(){
	time.Local = time.FixedZone("CST", 0)
	logFileLocation, _ := os.OpenFile("./log.info", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	mw := io.MultiWriter(os.Stdout, logFileLocation)
	myLog.SetOutput(mw)
}

func parsePending(node *node.Node, backend ethapi.Backend, eth *eth.Ethereum) {
	newTxCh := make(chan core.NewTxsEvent)
	backend.SubscribeNewTxsEvent(newTxCh)
	for {
		select {
		case tx := <-newTxCh:
			for _, txn := range tx.Txs {
				go simulateTx(txn, backend, eth)
			}
		}
	}
}

func simulateTx(txn *types.Transaction, backend ethapi.Backend, eth *eth.Ethereum) {
	// 合约创建和给value不为0 暂时都不考虑
	startTime := time.Now()
	if txn.To() == nil || txn.Value().Int64() != 0 || BlackContractAddress[strings.ToLower(txn.To().String())]{
		return
	}
	currentBN := backend.CurrentBlock().Number().Int64()
	fmt.Printf("current bn %d\n", currentBN)
	statedb, header, err := backend.StateAndHeaderByNumberOrHash(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(currentBN)))
	if statedb == nil || err != nil {
		fmt.Println("err get state")
		return
	}
	//对于每次模拟copy一份进行
	state := statedb.Copy()
	code := state.GetCode(*txn.To())
	//code为0是普通转账
	if len(code) == 0 {
		fmt.Printf("code len %d\n", len(code))
		return
	}
	//fmt.Println("================")
	//fmt.Printf("hash %s\n", txn.Hash().String())
	//fmt.Printf("value %s\n", txn.Value().String())
	//fmt.Printf("data %s\n", hex.EncodeToString(txn.Data()))
	//fmt.Printf("to %s\n", txn.To().String())
	//fmt.Printf("gasPrice %d\n", txn.GasPrice().Int64())
	//fmt.Printf("gas %d\n", txn.Gas())
	msg, e := txn.AsMessage(types.LatestSignerForChainID(txn.ChainId()))
	if e != nil {
		fmt.Println(e.Error())
	}
	from := msg.From().String()
	fmt.Printf("from %s\n", from)
	// 自己发的
	if strings.ToLower(from) == strings.ToLower(MyAddress) {
		return
	}
	// start to generate my simulated tx
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
	//fmt.Printf("direct sim receipt status %d\n", receipt.Status)
	if receipt.Status == 1 {
		logs := state.Logs()
		isProfitable := isProfitable(logs)
		if isProfitable {
			myLog.Printf("profit hash find ! txHash: %s", txn.Hash().String())
		}
		fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
	} /*else {
		// 直接模拟失败 则部署完全一样的合约进行模拟 若成功则抢跑部署+调用 若失败则放弃
		// 替换合约里写死的地址
		newCodeHexStr := strings.ReplaceAll(hex.EncodeToString(code), strings.ToLower(from[2:]), strings.ToLower(MyAddress[2:]))
		newCodeBytes, err := hex.DecodeString(newCodeHexStr)
		if err != nil {
			return
		}
		myCreateContractTx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce + 1,
			Value:    big.NewInt(0),
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
			fmt.Printf("logs %d\n", len(receiptCall.Logs))
			logs := state.Logs()
			for _, log := range logs {
				fmt.Printf("log address %s ", log.Address.String())
				for _, topic := range log.Topics {
					fmt.Printf("topic %s ", topic.String())
				}
				fmt.Println("")
			}
			fmt.Printf("time used %d ms\n", time.Since(startTime).Milliseconds())
		}
	}
	*/
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
				amount := binary.BigEndian.Uint64(log.Data)
				fmt.Printf("transfer amount %d(token %s) to me\n", amount, token)
				return true
			}
		}
	}
	return false
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node, backend ethapi.Backend) {
	debug.Memsize.Add("node", stack)
	fmt.Println("my custom node started..")
	// Start up the node itself
	utils.StartNode(ctx, stack)

	// Unlock any account specifically requested
	unlockAccounts(ctx, stack)

	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	// Create a client to interact with local geth node.
	rpcClient, err := stack.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to self: %v", err)
	}
	ethClient := ethclient.NewClient(rpcClient)

	go func() {
		// Open any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				var derivationPaths []accounts.DerivationPath
				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(derivationPaths, accounts.LegacyLedgerBaseDerivationPath)
				}
				derivationPaths = append(derivationPaths, accounts.DefaultBaseDerivationPath)

				event.Wallet.SelfDerive(derivationPaths, ethClient)

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()

	// Spawn a standalone goroutine for status synchronization monitoring,
	// close the node when synchronization is complete if user required.
	if ctx.GlobalBool(utils.ExitWhenSyncedFlag.Name) {
		go func() {
			sub := stack.EventMux().Subscribe(downloader.DoneEvent{})
			defer sub.Unsubscribe()
			for {
				event := <-sub.Chan()
				if event == nil {
					continue
				}
				done, ok := event.Data.(downloader.DoneEvent)
				if !ok {
					continue
				}
				if timestamp := time.Unix(int64(done.Latest.Time), 0); time.Since(timestamp) < 10*time.Minute {
					log.Info("Synchronisation completed", "latestnum", done.Latest.Number, "latesthash", done.Latest.Hash(),
						"age", common.PrettyAge(timestamp))
					stack.Close()
				}
			}
		}()
	}

	// Start auxiliary services if enabled
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) || ctx.GlobalBool(utils.DeveloperFlag.Name) {
		// Mining only makes sense if a full Ethereum node is running
		if ctx.GlobalString(utils.SyncModeFlag.Name) == "light" {
			utils.Fatalf("Light clients do not support mining")
		}
		ethBackend, ok := backend.(*eth.EthAPIBackend)
		if !ok {
			utils.Fatalf("Ethereum service not running: %v", err)
		}
		// Set the gas price to the limits from the CLI and start mining
		gasprice := utils.GlobalBig(ctx, utils.MinerGasPriceFlag.Name)
		ethBackend.TxPool().SetGasPrice(gasprice)
		// start mining
		threads := ctx.GlobalInt(utils.MinerThreadsFlag.Name)
		if err := ethBackend.StartMining(threads); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}
}

// unlockAccounts unlocks any account specifically requested.
func unlockAccounts(ctx *cli.Context, stack *node.Node) {
	var unlocks []string
	inputs := strings.Split(ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for _, input := range inputs {
		if trimmed := strings.TrimSpace(input); trimmed != "" {
			unlocks = append(unlocks, trimmed)
		}
	}
	// Short circuit if there is no account to unlock.
	if len(unlocks) == 0 {
		return
	}
	// If insecure account unlocking is not allowed if node's APIs are exposed to external.
	// Print warning log to user and skip unlocking.
	if !stack.Config().InsecureUnlockAllowed && stack.Config().ExtRPCEnabled() {
		utils.Fatalf("Account unlock with HTTP access is forbidden!")
	}
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	passwords := utils.MakePasswordList(ctx)
	for i, account := range unlocks {
		unlockAccount(ks, account, i, passwords)
	}
}
