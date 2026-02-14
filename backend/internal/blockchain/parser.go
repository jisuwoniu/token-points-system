package blockchain

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"token-points-system/internal/models"
)

// TransferEvent 表示解析后的Transfer事件
type TransferEvent struct {
	From     common.Address
	To       common.Address
	Value    *big.Int
	TxHash   string
	BlockNum int64
}

// ParseTransferLog 将区块链日志解析为TransferEvent
func ParseTransferLog(log types.Log) (*TransferEvent, error) {
	if len(log.Topics) < 3 {
		return nil, ErrInvalidLogFormat
	}

	from := common.BytesToAddress(log.Topics[1].Bytes())
	to := common.BytesToAddress(log.Topics[2].Bytes())

	value := new(big.Int)
	if len(log.Data) > 0 {
		value.SetBytes(log.Data)
	}

	return &TransferEvent{
		From:     from,
		To:       to,
		Value:    value,
		TxHash:   log.TxHash.Hex(),
		BlockNum: int64(log.BlockNumber),
	}, nil
}

// DetermineChangeType 确定用户的余额变动类型
// 返回：mint、burn或transfer
func (e *TransferEvent) DetermineChangeType(userAddress string) models.ChangeType {
	addr := strings.ToLower(userAddress)
	from := strings.ToLower(e.From.Hex())
	to := strings.ToLower(e.To.Hex())

	if from == "0x0000000000000000000000000000000000000000" {
		return models.ChangeTypeMint
	}
	if to == "0x0000000000000000000000000000000000000000" {
		return models.ChangeTypeBurn
	}
	if from == addr {
		return models.ChangeTypeTransfer
	}
	return models.ChangeTypeTransfer
}

// GetChangeAmount 获取用户的余额变动数量
// 发送方返回负数，接收方返回正数
func (e *TransferEvent) GetChangeAmount(userAddress string) *big.Int {
	addr := strings.ToLower(userAddress)
	from := strings.ToLower(e.From.Hex())

	if from == addr {
		return new(big.Int).Neg(e.Value)
	}
	return new(big.Int).Set(e.Value)
}

var ErrInvalidLogFormat = &InvalidLogFormatError{}

type InvalidLogFormatError struct{}

func (e *InvalidLogFormatError) Error() string {
	return "无效的日志格式：主题数量不足"
}
