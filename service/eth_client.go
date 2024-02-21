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
	"github.com/ethereum/go-ethereum/ethclient"
)

// SimpleEthClient exposes the eth_getBalance wrapper from the go-ethereum library
type SimpleEthClient interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) // queries eth balance at the specified block. If nil blockNumber is supplied the node will return the latest confirmed balance
	ethereum.BlockNumberReader                                                                     // Used for healthcheck/readiness probe
}

func NewEthClient(url string) (SimpleEthClient, error) {
	return ethclient.Dial(url)
}

// Multi nodes

var _ SimpleEthClient = (*multiNodeClient)(nil)

type multiNodeClient struct {
	nodes []*item
	mu    sync.RWMutex
}

// item has an idea so that when we update a node in the priority list, we sure that the priority list was not update before
type item struct {
	id     string // id is the position on the config url string
	client SimpleEthClient
}

// NewMultiNodeClient connects to a comma-separated list of ethereum clients
//
// TODO - more full description
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

// multiNodeCall is a generic pattern for any request RPC to multiple Ethereum clients that terminates
// at the first successful request. Any changes to node selection or prioritization logic
// should be made here.
func multiNodeCall[result any, request func() (string, result, error)](m *multiNodeClient, requests []request) (out result, err error) {
	for i := 0; i < len(requests); i++ {
		var id string
		id, out, err = requests[i]()
		if err == nil {
			m.increaseNodePriority(i, id)
			break
		}
	}
	return
}

// BalanceAt prepares a balance query to all nodes in the multiNodeClient set.
func (m *multiNodeClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var requests []func() (string, *big.Int, error)
	for i := 0; i < len(m.nodes); i++ {
		index := i
		m.mu.RLock()
		node := m.nodes[index]
		m.mu.RUnlock()
		requests = append(requests, func() (string, *big.Int, error) {
			res, err := node.client.BalanceAt(ctx, account, blockNumber)
			return node.id, res, err
		})
	}
	return multiNodeCall(m, requests)
}

const blockDiff = 3 // criteria for reporting failure based on two connected clients reporting different block numbers

func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// BlockNumber is used as part of the liveness probe for multiclients and will return
// and error if any of the connected clients report to be syncing.
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
				errStr += fmt.Sprintf("nodes %d (%d) and %d (%d) are reporting different chain tips|", index, index-1, f, s)
			}
		}
		m.mu.RUnlock()
	}
	if errStr != "" {
		return 0, errors.New(errStr)
	}
	return blockheights[0], nil
}
