package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"


	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 运行方式：
//
//	go run . --contract <contract address> --action number
//	go run . --contract <contract address> --action increment
//	go run . --contract <contract address> --action set --value 42
//
// 环境变量：
//
//	ETH_RPC_URL=https://sepolia.infura.io/v3/<projectId>
//	SENDER_PRIVATE_KEY=<your private key>
func main() {
	contractAddrFlag := flag.String("contract", "", "contract address on Sepolia")
	actionFlag := flag.String("action", "number", "action to run: number | increment | set")
	valueFlag := flag.Uint64("value", 0, "value for set action")
	flag.Parse()

	if *contractAddrFlag == "" {
		log.Fatal("--contract is required")
	}

	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}

	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" && *actionFlag != "number" {
		log.Fatal("SENDER_PRIVATE_KEY is not set; required for state-changing actions")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Sepolia node: %v", err)
	}
	defer client.Close()

	contractAddress := common.HexToAddress(*contractAddrFlag)
	contract, err := NewMain(contractAddress, client)
	if err != nil {
		log.Fatalf("failed to bind contract: %v", err)
	}

	switch strings.ToLower(*actionFlag) {
	case "number":
		current, err := contract.Number(nil)
		if err != nil {
			log.Fatalf("failed to read number: %v", err)
		}
		fmt.Printf("contract %s current number = %s\n", contractAddress.Hex(), current.String())

	case "increment":
		auth, err := newTransactOpts(ctx, client, privKeyHex)
		if err != nil {
			log.Fatalf("failed to create transact opts: %v", err)
		}
		tx, err := contract.Increment(auth)
		if err != nil {
			log.Fatalf("failed to send increment transaction: %v", err)
		}
		fmt.Printf("increment tx sent: %s\n", tx.Hash().Hex())

	case "set":
		auth, err := newTransactOpts(ctx, client, privKeyHex)
		if err != nil {
			log.Fatalf("failed to create transact opts: %v", err)
		}
		tx, err := contract.SetNumber(auth, new(big.Int).SetUint64(*valueFlag))
		if err != nil {
			log.Fatalf("failed to send setNumber transaction: %v", err)
		}
		fmt.Printf("setNumber tx sent: %s\n", tx.Hash().Hex())

	default:
		log.Fatalf("unsupported action: %s", *actionFlag)
	}
}

func newTransactOpts(ctx context.Context, client *ethclient.Client, privKeyHex string) (*bind.TransactOpts, error) {
	key, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain id: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}
	auth.Context = ctx

	tipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas tip cap: %w", err)
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read head header: %w", err)
	}
	baseFee := header.BaseFee
	if baseFee == nil {
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %w", err)
		}
		auth.GasPrice = gasPrice
	} else {
		feeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), tipCap)
		auth.GasTipCap = tipCap
		auth.GasFeeCap = feeCap
	}

	return auth, nil
}

func trim0x(s string) string {
	if strings.HasPrefix(s, "0x") {
		return s[2:]
	}
	return s
}
