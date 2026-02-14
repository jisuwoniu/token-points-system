package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"token-points-system/internal/config"
	"token-points-system/pkg/errors"
	"token-points-system/pkg/logger"
)

type Client struct {
	chainCfg *config.ChainConfig
	client   *ethclient.Client
}

// NewClient 创建指定链的区块链客户端
func NewClient(chainCfg *config.ChainConfig) (*Client, error) {
	client, err := ethclient.Dial(chainCfg.RPCURL)
	if err != nil {
		return nil, errors.New(errors.ErrRPConnect,
			fmt.Sprintf("连接RPC失败: %s", chainCfg.RPCURL), err)
	}

	return &Client{
		chainCfg: chainCfg,
		client:   client,
	}, nil
}

// Close 关闭区块链客户端连接
func (c *Client) Close() {
	c.client.Close()
}

// GetLatestBlockNumber 获取区块链最新区块号
func (c *Client) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	header, err := c.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, errors.New(errors.ErrBlockFetch, "获取最新区块失败", err)
	}
	return header.Number.Int64(), nil
}

// GetBlockByNumber 根据区块号获取区块
func (c *Client) GetBlockByNumber(ctx context.Context, number int64) (*types.Block, error) {
	block, err := c.client.BlockByNumber(ctx, big.NewInt(number))
	if err != nil {
		return nil, errors.New(errors.ErrBlockFetch,
			fmt.Sprintf("获取区块 %d 失败", number), err)
	}
	return block, nil
}

// GetConfirmBlockNumber 获取已确认的最新区块号
// 应用确认区块阈值后返回
func (c *Client) GetConfirmBlockNumber(ctx context.Context) (int64, error) {
	latest, err := c.GetLatestBlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	confirmed := latest - int64(c.chainCfg.ConfirmationBlocks)
	if confirmed < 0 {
		confirmed = 0
	}

	return confirmed, nil
}

// GetTransferLogs 获取指定区块范围内的Transfer事件日志
// 注意：RPC节点通常限制每次请求最多10,000个区块
func (c *Client) GetTransferLogs(ctx context.Context, startBlock, endBlock int64) ([]types.Log, error) {
	contractAddr := common.HexToAddress(c.chainCfg.ContractAddress)
	transferSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(startBlock),
		ToBlock:   big.NewInt(endBlock),
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{{transferSig}},
	}

	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, errors.New(errors.ErrEventParse, "过滤Transfer事件失败", err)
	}

	logger.WithFields(map[string]interface{}{
		"chain_id":    c.chainCfg.ID,
		"start_block": startBlock,
		"end_block":   endBlock,
		"logs_count":  len(logs),
	}).Info("获取Transfer事件日志")

	return logs, nil
}

// GetBlockTimestamp 获取区块的时间戳
func (c *Client) GetBlockTimestamp(ctx context.Context, blockNumber int64) (time.Time, error) {
	block, err := c.GetBlockByNumber(ctx, blockNumber)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(block.Time()), 0), nil
}
