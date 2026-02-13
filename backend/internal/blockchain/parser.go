package blockchain

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"token-points-system/internal/models"
)

type TransferEvent struct {
	From     common.Address
	To       common.Address
	Value    *big.Int
	TxHash   string
	BlockNum int64
}

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
		BlockNum: log.BlockNumber,
	}, nil
}

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
	return "invalid log format: insufficient topics"
}
