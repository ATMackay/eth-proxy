package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SimpleEthClient exposes the eth_getBalance wrapper from the go-ethereum library
type SimpleEthClient interface {
	ethereum.BlockNumberReader
	ethereum.TransactionReader
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) // queries eth balance at the specified block. If nil blockNumber is supplied the node will return the latest confirmed balance.
}

// NewEthClient wraps the connector to the goven URL
func NewEthClient(url string) (SimpleEthClient, error) {
	return ethclient.Dial(url)
}

var _ SimpleEthClient = (*multiNodeClient)(nil)

// Multi nodes

type multiNodeClient struct {
	nodes []*item
	mu    sync.RWMutex
}

// item is used to track the ordering of multiple eth RPC clients.
type item struct {
	id     string // id is the position on the config url string
	client SimpleEthClient
}

// NewMultiNodeClient connects to a comma-separated list of ethereum clients and stores them in an ordered
// list where they can be prioritized bad on the logic implemented in multiNodeCall.
func NewMultiNodeClient(possibleUrls string, constructor func(url string) (SimpleEthClient, error)) (*multiNodeClient, error) {
	urls := strings.Split(possibleUrls, ",")
	var nodes []*item
	errors := make(map[string]error)
	for i := 0; i < len(urls); i++ {
		url := urls[i]
		var node SimpleEthClient
		n, err := constructor(url)
		node = n
		if err != nil {
			errors[url] = err
			continue
		}
		nodes = append(nodes, &item{
			id:     fmt.Sprintf("%d", len(nodes)),
			client: node,
		})
	}
	if len(nodes) == 0 {
		message := "cannot connect to any nodes"
		for url, err := range errors {
			message = fmt.Sprintf("%s url='%s' err='%s'", message, url, err.Error())
		}
		return nil, fmt.Errorf(message)
	}
	return &multiNodeClient{
		nodes: nodes,
	}, nil
}

// increaseNodePriority bumps a client up one place in the slice.
func (m *multiNodeClient) increaseNodePriority(position int, id string) {
	if position == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodes[position].id != id {
		return
	}
	m.nodes[position-1], m.nodes[position] = m.nodes[position], m.nodes[position-1]
}

// BalanceAt prepares a balance query to all nodes in the multiNodeClient set.
func (m *multiNodeClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (bal *big.Int, err error) {
	for i := 0; i < len(m.nodes); i++ {
		index := i
		m.mu.RLock()
		node := m.nodes[index]
		bal, err = node.client.BalanceAt(ctx, account, blockNumber)
		m.mu.RUnlock()
		if err == nil {
			m.increaseNodePriority(i, node.id)
			break
		}
	}
	return
}

const blockDiff = 3 // criteria for reporting failure based on two connected clients reporting different block numbers

func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// BlockNumber is used as part of the liveness probe for multiNodeClient and will return
// and error if the connected Ethereum nodes report block heights with disparity grater
// then the blockDiff limit.
func (m *multiNodeClient) BlockNumber(ctx context.Context) (uint64, error) {
	var blockheights []uint64
	var errStr string
	for i := 0; i < len(m.nodes); i++ {
		index := i
		m.mu.RLock()
		node := m.nodes[index]
		b, err := node.client.BlockNumber(ctx)
		if err != nil {
			errStr += fmt.Sprintf("node %d err: %s|", index, err.Error())
			m.mu.RUnlock()
			continue
		}
		blockheights = append(blockheights, b)
		if len(blockheights) > 1 {
			if f, s := blockheights[len(blockheights)-1], blockheights[len(blockheights)-2]; absDiff(f, s) > blockDiff {
				errStr += fmt.Sprintf("nodes %d (height=%d) and %d (height=%d) are reporting different chain tips|", index, f, index-1, s)
			}
		}
		m.mu.RUnlock()
	}
	if errStr != "" {
		return 0, errors.New(errStr)
	}
	return blockheights[0], nil
}

// TransactionByHash checks the pool of pending transactions in addition to the
// blockchain. The isPending return value indicates whether the transaction has been
// mined yet. Note that the transaction may not be part of the canonical chain even if
// it's not pending.
func (m *multiNodeClient) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	for i := 0; i < len(m.nodes); i++ {
		index := i
		m.mu.RLock()
		node := m.nodes[index]
		tx, isPending, err = node.client.TransactionByHash(ctx, txHash)
		m.mu.RUnlock()
		if err == nil {
			m.increaseNodePriority(i, node.id)
			break
		}
	}
	return
}

// TransactionReceipt returns the receipt of a mined transaction. Note that the
// transaction may not be included in the current canonical chain even if a receipt
// exists.
func (m *multiNodeClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (receipt *types.Receipt, err error) {
	for i := 0; i < len(m.nodes); i++ {
		index := i
		m.mu.RLock()
		node := m.nodes[index]
		receipt, err = node.client.TransactionReceipt(ctx, txHash)
		m.mu.RUnlock()
		if err == nil {
			m.increaseNodePriority(i, node.id)
			break
		}
	}
	return
}
